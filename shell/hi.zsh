# hi-shell zsh plugin

if [[ -n "${_HI_SHELL_LOADED:-}" ]]; then
  return
fi
_HI_SHELL_LOADED=1

hi() {
  print -u2 "hi-shell: 'hi' is a zsh prompt prefix; use 'hi-shell ...' for CLI commands"
  return 2
}

typeset -g _HI_ORIGINAL_BUFFER=""
typeset -g _HI_ORIGINAL_PROMPT=""
typeset -g _HI_SUGGESTION=""
typeset -g _HI_RISK=""
typeset -g _HI_WARNING=""
typeset -g _HI_REFINING=""
typeset -g _HI_ENTER_WIDGET=""
typeset -g _HI_LINEFEED_WIDGET=""
typeset -g _HI_TAB_WIDGET=""
typeset -g _HI_CTRL_C_WIDGET=""
typeset -g _HI_SELF_INSERT_WIDGET="_hi_original_self_insert"
typeset -ga _HI_TURN_COMMANDS=()
typeset -ga _HI_TURN_RISKS=()
typeset -ga _HI_TURN_WARNINGS=()
typeset -ga _HI_TURN_FEEDBACKS=()

autoload -Uz add-zle-hook-widget

_hi_bound_widget() {
  local binding
  binding="$(bindkey "$1" 2>/dev/null)" || return
  print -r -- "${binding##* }"
}

_hi_call_widget() {
  local widget="$1"
  local fallback="$2"

  if [[ -n "$widget" && "$widget" != _hi_accept_line && "$widget" != _hi_accept_suggestion && "$widget" != _hi_cancel && "$widget" != _hi_self_insert ]]; then
    zle "$widget" 2>/dev/null && return
  fi

  zle "$fallback"
}

_hi_clear_state() {
  _HI_ORIGINAL_BUFFER=""
  _HI_ORIGINAL_PROMPT=""
  _HI_SUGGESTION=""
  _HI_RISK=""
  _HI_WARNING=""
  _HI_REFINING=""
  _HI_TURN_COMMANDS=()
  _HI_TURN_RISKS=()
  _HI_TURN_WARNINGS=()
  _HI_TURN_FEEDBACKS=()
  POSTDISPLAY=""
}

_hi_json_string() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  value="${value//$'\n'/\\n}"
  value="${value//$'\r'/\\r}"
  value="${value//$'\t'/\\t}"
  print -r -- "\"$value\""
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
  local count="${#_HI_TURN_COMMANDS[@]}"
  local i
  local wrote_turn=0

  print -rn -- '{"initial_prompt":'
  _hi_json_string "$_HI_ORIGINAL_PROMPT"
  print -rn -- ',"turns":['

  for (( i = 1; i <= count; i++ )); do
    if (( wrote_turn )); then
      print -rn -- ','
    fi
    _hi_turn_json "${_HI_TURN_COMMANDS[$i]}" "${_HI_TURN_RISKS[$i]}" "${_HI_TURN_WARNINGS[$i]}" "${_HI_TURN_FEEDBACKS[$i]}"
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

_hi_generate_for_prompt() {
  local query="$1"
  local result command risk warning

  _hi_clear_state

  result="$(command hi-shell generate --prompt "$query" --format json 2>&1)"
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
    zle redisplay
    if [[ -n "$warning" ]]; then
      zle -M "hi-shell: $warning"
    else
      zle -M "hi-shell: no command generated"
    fi
    return 1
  fi

  _HI_ORIGINAL_PROMPT="$query"
  _HI_SUGGESTION="$command"
  _HI_RISK="$risk"
  _HI_WARNING="$warning"
  _HI_REFINING=""

  BUFFER=""
  CURSOR=0
  POSTDISPLAY="$_HI_SUGGESTION"

  zle redisplay

  if [[ -n "$_HI_WARNING" && "$_HI_RISK" == (high|critical) ]]; then
    zle -M "hi-shell: $_HI_WARNING"
  fi
}

_hi_revise_for_feedback() {
  local feedback="$1"
  local result command risk warning

  result="$(_hi_revision_json "$feedback" | command hi-shell revise --session-json - --format json 2>&1)"
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
    zle redisplay
    if [[ -n "$warning" ]]; then
      zle -M "hi-shell: $warning"
    else
      zle -M "hi-shell: no command generated"
    fi
    return 1
  fi

  _HI_TURN_COMMANDS+=("$_HI_SUGGESTION")
  _HI_TURN_RISKS+=("$_HI_RISK")
  _HI_TURN_WARNINGS+=("$_HI_WARNING")
  _HI_TURN_FEEDBACKS+=("$feedback")

  _HI_SUGGESTION="$command"
  _HI_RISK="$risk"
  _HI_WARNING="$warning"
  _HI_REFINING=""

  BUFFER=""
  CURSOR=0
  POSTDISPLAY="$_HI_SUGGESTION"

  zle redisplay

  if [[ -n "$_HI_WARNING" && "$_HI_RISK" == (high|critical) ]]; then
    zle -M "hi-shell: $_HI_WARNING"
  fi
}

_hi_accept_line() {
  local widget="$_HI_ENTER_WIDGET"

  if [[ -n "$_HI_SUGGESTION" ]]; then
    local feedback="$BUFFER"

    if [[ -z "${feedback//[[:space:]]/}" ]]; then
      zle redisplay
      zle -M "hi-shell: press Tab to accept or type feedback"
      return
    fi

    _hi_revise_for_feedback "$feedback"
    return
  fi

  if [[ "$BUFFER" == hi\ * ]]; then
    local query="${BUFFER#hi }"
    _HI_ORIGINAL_BUFFER="$BUFFER"
    _hi_generate_for_prompt "$query"
    return
  fi

  if [[ "$KEYS" == $'\n' ]]; then
    widget="$_HI_LINEFEED_WIDGET"
  fi

  _hi_clear_state
  _hi_call_widget "$widget" ".accept-line"
}

_hi_accept_suggestion() {
  if [[ -n "$_HI_SUGGESTION" && -z "$_HI_REFINING" ]]; then
    BUFFER="$_HI_SUGGESTION"
    CURSOR=${#BUFFER}
    _hi_clear_state
    zle redisplay
    return
  fi

  _hi_call_widget "$_HI_TAB_WIDGET" ".expand-or-complete"
}

_hi_self_insert() {
  if [[ -n "$_HI_SUGGESTION" && -z "$_HI_REFINING" ]]; then
    POSTDISPLAY=""
    _HI_REFINING=1
  fi

  _hi_call_widget "$_HI_SELF_INSERT_WIDGET" ".self-insert"
}

_hi_line_pre_redraw() {
  if [[ -n "$_HI_REFINING" ]]; then
    region_highlight=()
  fi
}

_hi_cancel() {
  if [[ -n "$_HI_SUGGESTION" || -n "$_HI_ORIGINAL_BUFFER" ]]; then
    BUFFER=""
    CURSOR=0
    _hi_clear_state
    zle redisplay
    return
  fi

  _hi_call_widget "$_HI_CTRL_C_WIDGET" ".send-break"
}

_HI_ENTER_WIDGET="$(_hi_bound_widget '^M')"
_HI_LINEFEED_WIDGET="$(_hi_bound_widget '^J')"
_HI_TAB_WIDGET="$(_hi_bound_widget '^I')"
_HI_CTRL_C_WIDGET="$(_hi_bound_widget '^C')"

zle -N _hi_accept_line
zle -N _hi_accept_suggestion
zle -A self-insert "$_HI_SELF_INSERT_WIDGET"
zle -N self-insert _hi_self_insert
zle -N _hi_cancel

add-zle-hook-widget zle-line-pre-redraw _hi_line_pre_redraw 2>/dev/null

bindkey '^M' _hi_accept_line
bindkey '^J' _hi_accept_line
bindkey '^I' _hi_accept_suggestion
bindkey '^C' _hi_cancel
