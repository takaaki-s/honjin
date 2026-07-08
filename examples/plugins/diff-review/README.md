# diff-review

A reference jin plugin: opens a tmux popup showing `git diff` for a session's
working directory, lets you pick out the lines you want to comment on, and
sends the assembled feedback back into the session as a prompt.

This is an **action** plugin (`on: []` in its manifest) — it never fires on a
`status_changed` event. You launch it explicitly against a session.

## What it demonstrates

- **The popup hand-off contract.** `jin pane popup` runs its command in a
  fresh process that tmux spawns — that process does not inherit this
  plugin's `JIN_*` environment. The outer stage bakes `JIN_SESSION_ID` and
  `JIN_WORKDIR` into the popup's command line instead of relying on env vars.
  See the comments in `diff-review.sh` for the two-stage (`--inner`) layout
  this requires.
- **Graceful degradation.** Falls back to a single overall comment when
  `fzf` isn't installed, and to `less` when `delta` isn't installed.

## Install

```bash
jin plugin install --link /path/to/jindaiko/examples/plugins/diff-review
```

`--link` symlinks this directory into place, so editing `diff-review.sh`
here takes effect on the next run without reinstalling.

## Run

```bash
jin plugin run diff-review --session <selector>
```

`<selector>` is a session ID prefix or a description substring, same as
`jin session` commands. This opens a tmux popup anchored to that session's
pane; the session's status is untouched (this plugin never sends anything
unless you enter feedback).

## Dependencies

| Tool | Required? | Effect if missing |
|------|-----------|--------------------|
| `git` | Yes | The plugin exits with an error before opening the popup |
| `fzf` | No | Falls back to a single "overall feedback" prompt instead of per-line comments |
| `delta` | No | Falls back to plain `git diff` output paged through `less` |
