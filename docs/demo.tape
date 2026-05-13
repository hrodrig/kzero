# kzero terminal demo — from repository root, with kzero on PATH:
#   make build && export PATH="$(pwd)/bin:$PATH" && bash -c "vhs docs/demo.tape"
# Or: make install && bash -c "vhs docs/demo.tape"
# Use bash -c (not bare `vhs` from zsh) so the recorder does not inherit zsh/Oh My Zsh prompts.
# Re-record after VERSION changes if the tape shows `kzero version` output.

Output docs/demo.gif

Require kzero

Set Shell "bash"
Set FontSize 14
Set Width 920
Set Height 520
Set Padding 20

Enter
Sleep 300ms

# Help
Type "kzero --help"
Enter
Sleep 2s

# Version (ldflags when built via make build / make install)
Type "kzero version"
Enter
Sleep 1s

# Analyze minimal demo config (no cluster mutations)
Type "kzero analyze --config docs/demo-kzero.yaml"
Enter
Sleep 2s

# Clean exit
Type "exit"
Enter
Sleep 500ms
