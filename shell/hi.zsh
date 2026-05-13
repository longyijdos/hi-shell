# hi-shell zsh plugin

if [[ -n "${_HI_SHELL_LOADED:-}" ]]; then
  return
fi
_HI_SHELL_LOADED=1

hi() {
  print -u2 "hi-shell: 'hi' is a zsh prompt prefix; use 'hi-shell ...' for CLI commands"
  return 2
}

typeset -g _HI_ORIGINAL_PROMPT=""
typeset -g _HI_SUGGESTION=""
typeset -g _HI_RISK=""
typeset -g _HI_WARNING=""
typeset -g _HI_MODE="edit"
typeset -g _HI_ENTER_WIDGET=""
typeset -g _HI_LINEFEED_WIDGET=""
typeset -g _HI_TAB_WIDGET=""
typeset -g _HI_PREFIX_KEY=""
typeset -g _HI_PREFIX_WIDGET=""
typeset -g _HI_SELF_INSERT_WIDGET="_hi_original_self_insert"
typeset -g _HI_HIGHLIGHT_DISABLED=""
typeset -g _HI_ZSH_HIGHLIGHT_HIGHLIGHTERS_SET=""
typeset -gi _HI_HISTORY_FETCH_LIMIT=0
typeset -ga _HI_REVISE_TURNS=()
typeset -ga _HI_ASK_TURNS=()
typeset -ga _HI_ZSH_HIGHLIGHT_HIGHLIGHTERS=()

autoload -Uz add-zle-hook-widget add-zsh-hook

_hi_bound_widget() {
  local binding
  binding="$(bindkey "$1" 2>/dev/null)" || return
  print -r -- "${binding##* }"
}

_hi_call_widget() {
  local widget="$1"
  local fallback="$2"

  case "$widget" in
    ""|_hi_accept_line|_hi_accept_suggestion|_hi_prefix_*|_hi_self_insert) ;;
    *) zle "$widget" 2>/dev/null && return ;;
  esac

  zle "$fallback"
}

_hi_config_get() {
  local key="$1"
  local default_value="$2"
  local value

  if command -v hi-shell >/dev/null 2>&1; then
    value="$(command hi-shell config get "$key" 2>/dev/null)"
  fi
  value="${value//$'\n'/}"

  if [[ -z "$value" ]]; then
    value="$default_value"
  fi

  print -r -- "$value"
}

_hi_clear_state() {
  _hi_set_mode edit
  _HI_ORIGINAL_PROMPT=""
  _HI_SUGGESTION=""
  _HI_RISK=""
  _HI_WARNING=""
  _HI_REVISE_TURNS=()
  _HI_ASK_TURNS=()
  POSTDISPLAY=""
}

_hi_disable_highlighting() {
  if [[ -z "$_HI_HIGHLIGHT_DISABLED" ]]; then
    _HI_HIGHLIGHT_DISABLED=1
    if (( ${+ZSH_HIGHLIGHT_HIGHLIGHTERS} )); then
      _HI_ZSH_HIGHLIGHT_HIGHLIGHTERS_SET=1
      _HI_ZSH_HIGHLIGHT_HIGHLIGHTERS=("${ZSH_HIGHLIGHT_HIGHLIGHTERS[@]}")
    else
      _HI_ZSH_HIGHLIGHT_HIGHLIGHTERS_SET=""
      _HI_ZSH_HIGHLIGHT_HIGHLIGHTERS=()
    fi
    ZSH_HIGHLIGHT_HIGHLIGHTERS=()
  fi

  region_highlight=()
  zle redisplay
}

_hi_restore_highlighting() {
  if [[ -z "$_HI_HIGHLIGHT_DISABLED" ]]; then
    return
  fi

  if [[ -n "$_HI_ZSH_HIGHLIGHT_HIGHLIGHTERS_SET" ]]; then
    ZSH_HIGHLIGHT_HIGHLIGHTERS=("${_HI_ZSH_HIGHLIGHT_HIGHLIGHTERS[@]}")
  else
    unset ZSH_HIGHLIGHT_HIGHLIGHTERS
  fi

  _HI_HIGHLIGHT_DISABLED=""
  _HI_ZSH_HIGHLIGHT_HIGHLIGHTERS_SET=""
  _HI_ZSH_HIGHLIGHT_HIGHLIGHTERS=()
  region_highlight=()
}

_hi_set_mode() {
  _HI_MODE="$1"
  case "$_HI_MODE" in
    revise|ask) _hi_disable_highlighting ;;
    *) _HI_MODE="edit"; _hi_restore_highlighting ;;
  esac
}

_hi_show_warning() {
  local warning="$1"

  zle redisplay
  if [[ -n "$warning" ]]; then
    zle -M "hi-shell: $warning"
  else
    zle -M "hi-shell: no command generated"
  fi
}

_hi_recent_history() {
  fc -ln "-${_HI_HISTORY_FETCH_LIMIT}" 2>/dev/null
}

_hi_run_cli() {
  local recent_history

  recent_history="$(_hi_recent_history)"
  if [[ -n "$recent_history" ]]; then
    HI_SHELL_HISTORY="$recent_history" command hi-shell "$@"
    return
  fi

  command hi-shell "$@"
}

_hi_display_suggestion() {
  _HI_SUGGESTION="$1"
  _HI_RISK="$2"
  _HI_WARNING="$3"
  _hi_set_mode edit

  BUFFER=""
  CURSOR=0
  POSTDISPLAY="$_HI_SUGGESTION"

  zle redisplay

  if [[ -n "$_HI_WARNING" ]]; then
    zle -M "hi-shell: $_HI_WARNING"
  fi
}

_hi_json_string() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  value="${value//$'\n'/\\n}"
  value="${value//$'\r'/\\r}"
  value="${value//$'\t'/\\t}"
  print -rn -- "\"$value\""
}

_hi_turn_json() {
  local command="$1"
  local risk="$2"
  local warning="$3"
  local feedback="$4"

  print -rn -- '{"command":'
  _hi_json_string "$command"
  print -rn -- ',"risk":'
  _hi_json_string "$risk"
  print -rn -- ',"warning":'
  _hi_json_string "$warning"
  print -rn -- ',"feedback":'
  _hi_json_string "$feedback"
  print -rn -- '}'
}

_hi_revision_json() {
  local feedback="$1"
  local count="${#_HI_REVISE_TURNS[@]}"
  local i
  local wrote_turn=0

  print -rn -- '{"initial_prompt":'
  _hi_json_string "$_HI_ORIGINAL_PROMPT"
  print -rn -- ',"turns":['

  for (( i = 1; i <= count; i++ )); do
    if (( wrote_turn )); then
      print -rn -- ','
    fi
    print -rn -- "${_HI_REVISE_TURNS[$i]}"
    wrote_turn=1
  done

  if [[ -n "$_HI_SUGGESTION" ]]; then
    if (( wrote_turn )); then
      print -rn -- ','
    fi
    _hi_turn_json "$_HI_SUGGESTION" "$_HI_RISK" "$_HI_WARNING" "$feedback"
  fi

  print -r -- ']}'
}

_hi_ask_turn_json() {
  local command="$1"
  local risk="$2"
  local warning="$3"
  local question="$4"
  local answer="$5"

  print -rn -- '{"command":'
  _hi_json_string "$command"
  print -rn -- ',"risk":'
  _hi_json_string "$risk"
  print -rn -- ',"warning":'
  _hi_json_string "$warning"
  print -rn -- ',"question":'
  _hi_json_string "$question"
  print -rn -- ',"answer":'
  _hi_json_string "$answer"
  print -rn -- '}'
}

_hi_ask_json() {
  local question="$1"
  local count="${#_HI_ASK_TURNS[@]}"
  local i
  local wrote_turn=0

  print -rn -- '{"initial_prompt":'
  _hi_json_string "$_HI_ORIGINAL_PROMPT"
  print -rn -- ',"turns":['

  for (( i = 1; i <= count; i++ )); do
    if (( wrote_turn )); then
      print -rn -- ','
    fi
    print -rn -- "${_HI_ASK_TURNS[$i]}"
    wrote_turn=1
  done

  if [[ -n "$_HI_SUGGESTION" ]]; then
    if (( wrote_turn )); then
      print -rn -- ','
    fi
    _hi_ask_turn_json "$_HI_SUGGESTION" "$_HI_RISK" "$_HI_WARNING" "$question" ""
  fi

  print -r -- ']}'
}

_hi_generate_for_prompt() {
  local query="$1"
  local result command risk warning

  _hi_clear_state
  zle -R "hi-shell: generating..."

  result="$(_hi_run_cli generate --prompt "$query" --format json 2>&1)"
  if (( $? != 0 )); then
    _hi_clear_state
    zle redisplay
    zle -M "hi-shell: $result"
    return 1
  fi

  command="$(command hi-shell parse-field command "$result" 2>/dev/null)"
  risk="$(command hi-shell parse-field risk "$result" 2>/dev/null)"
  warning="$(command hi-shell parse-field warning "$result" 2>/dev/null)"

  if [[ -z "$command" ]]; then
    _hi_clear_state
    _hi_show_warning "$warning"
    return 1
  fi

  _HI_ORIGINAL_PROMPT="$query"
  _hi_display_suggestion "$command" "$risk" "$warning"
}

_hi_revise_for_feedback() {
  local feedback="$1"
  local result command risk warning

  zle -R "hi-shell: revising..."

  result="$(_hi_revision_json "$feedback" | _hi_run_cli revise --session-json - --format json 2>&1)"
  if (( $? != 0 )); then
    POSTDISPLAY=""
    zle redisplay
    zle -M "hi-shell: $result"
    return 1
  fi

  command="$(command hi-shell parse-field command "$result" 2>/dev/null)"
  risk="$(command hi-shell parse-field risk "$result" 2>/dev/null)"
  warning="$(command hi-shell parse-field warning "$result" 2>/dev/null)"

  if [[ -z "$command" ]]; then
    POSTDISPLAY=""
    _hi_show_warning "$warning"
    return 1
  fi

  _HI_REVISE_TURNS+=("$(_hi_turn_json "$_HI_SUGGESTION" "$_HI_RISK" "$_HI_WARNING" "$feedback")")
  _hi_display_suggestion "$command" "$risk" "$warning"
}

_hi_ask_question() {
  local question="$1"
  local result answer message

  zle -R "hi-shell: asking..."

  result="$(_hi_ask_json "$question" | _hi_run_cli ask --session-json - --format json 2>&1)"
  if (( $? != 0 )); then
    POSTDISPLAY=""
    zle redisplay
    zle -M "hi-shell: $result"
    return 1
  fi

  answer="$(command hi-shell parse-field answer "$result" 2>/dev/null)"
  if [[ -z "$answer" ]]; then
    POSTDISPLAY=""
    zle redisplay
    zle -M "hi-shell: no answer generated"
    return 1
  fi

  _HI_ASK_TURNS+=("$(_hi_ask_turn_json "$_HI_SUGGESTION" "$_HI_RISK" "$_HI_WARNING" "$question" "$answer")")
  _hi_set_mode edit
  BUFFER=""
  CURSOR=0
  POSTDISPLAY="$_HI_SUGGESTION"
  zle redisplay

  message="${answer//$'\n'/ }"
  zle -M "hi-shell: $message"
}

_hi_accept_line() {
  local widget="$_HI_ENTER_WIDGET"

  if [[ "$KEYS" == $'\n' ]]; then
    widget="$_HI_LINEFEED_WIDGET"
  fi

  if [[ -n "$_HI_SUGGESTION" && "$_HI_MODE" == "revise" ]]; then
    local feedback="$BUFFER"

    if [[ -z "${feedback//[[:space:]]/}" ]]; then
      zle redisplay
      zle -M "hi-shell: type feedback or press Ctrl-C to cancel"
      return
    fi

    _hi_revise_for_feedback "$feedback"
    return
  fi

  if [[ -n "$_HI_SUGGESTION" && "$_HI_MODE" == "ask" ]]; then
    local question="$BUFFER"

    if [[ -z "${question//[[:space:]]/}" ]]; then
      zle redisplay
      zle -M "hi-shell: type a question or press Ctrl-C to cancel"
      return
    fi

    _hi_ask_question "$question"
    return
  fi

  if [[ "$BUFFER" == hi\ * ]]; then
    local query="${BUFFER#hi }"
    _hi_generate_for_prompt "$query"
    return
  fi

  if [[ -n "$_HI_SUGGESTION" ]]; then
    _hi_clear_state
    _hi_call_widget "$widget" ".accept-line"
    return
  fi

  _hi_clear_state
  _hi_call_widget "$widget" ".accept-line"
}

_hi_accept_suggestion() {
  if [[ -n "$_HI_SUGGESTION" && -n "$POSTDISPLAY" ]]; then
    BUFFER="$_HI_SUGGESTION"
    CURSOR=${#BUFFER}
    POSTDISPLAY=""
    zle redisplay
    return
  fi

  _hi_call_widget "$_HI_TAB_WIDGET" ".expand-or-complete"
}

_hi_self_insert() {
  if [[ -n "$_HI_SUGGESTION" && -n "$POSTDISPLAY" ]]; then
    POSTDISPLAY=""
  fi

  _hi_call_widget "$_HI_SELF_INSERT_WIDGET" ".self-insert"
}

_hi_prefix_mode() {
  local mode="$1"
  local missing_message="$2"

  zle -K main

  if [[ -z "$_HI_SUGGESTION" ]]; then
    zle redisplay
    zle -M "$missing_message"
    return
  fi

  if [[ "$_HI_MODE" == "$mode" ]]; then
    zle redisplay
    zle -M "hi-shell: already in $mode mode"
    return
  fi

  _hi_set_mode "$mode"
  zle redisplay
  zle -M "hi-shell: $mode mode"
}

_hi_prefix_revise() {
  _hi_prefix_mode revise "hi-shell: no suggestion to revise"
}

_hi_prefix_edit() {
  _hi_prefix_mode edit "hi-shell: no suggestion to edit"
}

_hi_prefix_ask() {
  _hi_prefix_mode ask "hi-shell: no suggestion to ask about"
}

_hi_prefix_enter() {
  if [[ -z "$_HI_SUGGESTION" ]]; then
    _hi_call_widget "$_HI_PREFIX_WIDGET" ".undefined-key"
    return
  fi

  zle -K hi-shell-prefix
  zle -M "hi-shell: e edit, r revise, a ask, q exit"
}

_hi_prefix_cancel() {
  zle -K main
  zle redisplay
  zle -M ""
}

_hi_prefix_unknown() {
  zle -K main
  zle redisplay
  zle -M "hi-shell: unknown prefix key"
}

_HI_ENTER_WIDGET="$(_hi_bound_widget '^M')"
_HI_LINEFEED_WIDGET="$(_hi_bound_widget '^J')"
_HI_TAB_WIDGET="$(_hi_bound_widget '^I')"
_HI_PREFIX_KEY="$(_hi_config_get keybindings.prefix '^]')"
_HI_PREFIX_WIDGET="$(_hi_bound_widget "$_HI_PREFIX_KEY")"
_HI_HISTORY_FETCH_LIMIT="$(_hi_config_get history.fetch_limit 20)"

zle -N _hi_accept_line
zle -N _hi_accept_suggestion
zle -A self-insert "$_HI_SELF_INSERT_WIDGET"
zle -N self-insert _hi_self_insert
zle -N _hi_prefix_enter
zle -N _hi_prefix_revise
zle -N _hi_prefix_edit
zle -N _hi_prefix_ask
zle -N _hi_prefix_cancel
zle -N _hi_prefix_unknown

bindkey -N hi-shell-prefix
bindkey -M hi-shell-prefix -R "^@-^?" _hi_prefix_unknown
bindkey -M hi-shell-prefix 'e' _hi_prefix_edit
bindkey -M hi-shell-prefix 'r' _hi_prefix_revise
bindkey -M hi-shell-prefix 'a' _hi_prefix_ask
bindkey -M hi-shell-prefix 'q' _hi_prefix_cancel

add-zle-hook-widget zle-line-init _hi_clear_state 2>/dev/null
add-zle-hook-widget zle-line-finish _hi_clear_state 2>/dev/null
add-zsh-hook precmd _hi_clear_state 2>/dev/null

bindkey '^M' _hi_accept_line
bindkey '^J' _hi_accept_line
bindkey '^I' _hi_accept_suggestion
bindkey "$_HI_PREFIX_KEY" _hi_prefix_enter
