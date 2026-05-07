# hi-shell zsh plugin

if [[ -n "${_HI_SHELL_LOADED:-}" ]]; then
  return
fi
_HI_SHELL_LOADED=1

typeset -g _HI_ORIGINAL_BUFFER=""
typeset -g _HI_ORIGINAL_PROMPT=""
typeset -g _HI_SUGGESTION=""
typeset -g _HI_RISK=""
typeset -g _HI_WARNING=""
typeset -g _HI_TAB_WIDGET=""
typeset -g _HI_CTRL_C_WIDGET=""
typeset -g _HI_ESC_WIDGET=""
typeset -g _HI_CTRL_R_WIDGET=""

_hi_bound_widget() {
  local binding
  binding="$(bindkey "$1" 2>/dev/null)" || return
  print -r -- "${binding##* }"
}

_hi_call_widget() {
  local widget="$1"
  local fallback="$2"

  if [[ -n "$widget" && "$widget" != _hi_accept_suggestion && "$widget" != _hi_cancel && "$widget" != _hi_escape && "$widget" != _hi_regenerate ]]; then
    zle "$widget"
    return
  fi

  zle "$fallback"
}

_hi_clear_state() {
  _HI_ORIGINAL_BUFFER=""
  _HI_ORIGINAL_PROMPT=""
  _HI_SUGGESTION=""
  _HI_RISK=""
  _HI_WARNING=""
  POSTDISPLAY=""
}

_hi_generate_for_prompt() {
  local query="$1"
  local result command risk warning

  result="$(command hi generate --prompt "$query" --format json 2>&1)"
  if (( $? != 0 )); then
    _hi_clear_state
    zle -M "hi-shell: $result"
    zle redisplay
    return 1
  fi

  command="$(command hi parse-field command "$result" 2>/dev/null)"
  risk="$(command hi parse-field risk "$result" 2>/dev/null)"
  warning="$(command hi parse-field warning "$result" 2>/dev/null)"

  if [[ -z "$command" ]]; then
    _hi_clear_state
    if [[ -n "$warning" ]]; then
      zle -M "hi-shell: $warning"
    else
      zle -M "hi-shell: no command generated"
    fi
    zle redisplay
    return 1
  fi

  _HI_ORIGINAL_PROMPT="$query"
  _HI_SUGGESTION="$command"
  _HI_RISK="$risk"
  _HI_WARNING="$warning"

  BUFFER=""
  CURSOR=0
  POSTDISPLAY="$_HI_SUGGESTION"

  if [[ -n "$_HI_WARNING" && "$_HI_RISK" == (high|critical) ]]; then
    zle -M "hi-shell: $_HI_WARNING"
  fi

  zle redisplay
}

_hi_is_cli_subcommand() {
  local query="$1"
  local -a words

  words=(${(z)query})
  case "${words[1]:-}" in
    ""|generate|config|install|uninstall|doctor|version|parse-field|parse-command|help|-h|--help)
      return 0
      ;;
  esac

  return 1
}

_hi_accept_line() {
  if [[ "$BUFFER" == hi\ * ]]; then
    local query="${BUFFER#hi }"
    if _hi_is_cli_subcommand "$query"; then
      _hi_clear_state
      zle .accept-line
      return
    fi

    _HI_ORIGINAL_BUFFER="$BUFFER"
    _hi_generate_for_prompt "$query"
    return
  fi

  _hi_clear_state
  zle .accept-line
}

_hi_accept_suggestion() {
  if [[ -n "$_HI_SUGGESTION" ]]; then
    BUFFER="$_HI_SUGGESTION"
    CURSOR=${#BUFFER}
    _hi_clear_state
    zle redisplay
    return
  fi

  _hi_call_widget "$_HI_TAB_WIDGET" ".expand-or-complete"
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

_hi_escape() {
  if [[ -n "$_HI_SUGGESTION" ]]; then
    BUFFER=""
    CURSOR=0
    _hi_clear_state
    zle redisplay
    return
  fi

  _hi_call_widget "$_HI_ESC_WIDGET" ".undefined-key"
}

_hi_regenerate() {
  if [[ -n "$_HI_ORIGINAL_PROMPT" ]]; then
    _hi_generate_for_prompt "$_HI_ORIGINAL_PROMPT"
    return
  fi

  _hi_call_widget "$_HI_CTRL_R_WIDGET" ".history-incremental-search-backward"
}

_HI_TAB_WIDGET="$(_hi_bound_widget '^I')"
_HI_CTRL_C_WIDGET="$(_hi_bound_widget '^C')"
_HI_ESC_WIDGET="$(_hi_bound_widget '^[')"
_HI_CTRL_R_WIDGET="$(_hi_bound_widget '^R')"

zle -N accept-line _hi_accept_line
zle -N _hi_accept_suggestion
zle -N _hi_cancel
zle -N _hi_escape
zle -N _hi_regenerate

bindkey '^I' _hi_accept_suggestion
bindkey '^C' _hi_cancel
bindkey '^[' _hi_escape
bindkey '^R' _hi_regenerate
