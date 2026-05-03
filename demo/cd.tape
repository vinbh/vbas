Output demo/cd.gif

Set Shell "zsh"
Set FontSize 15
Set Width 860
Set Height 340
Set Theme "Dracula"
Set Padding 20
Set TypingSpeed 90ms
Set PlaybackSpeed 1.0

Hide
Type "mkdir -p /tmp/vbas-demo/projects/{myapp/{src,tests},backend,frontend} /tmp/vbas-demo/{notes,downloads}"
Enter
Type "cd /tmp/vbas-demo"
Enter
Type "export VBAS_SPECS_DIR=/home/vb666/.config/vbas/specs VBAS_BIN=/home/vb666/.local/bin/vbas"
Enter
Type "source /home/vb666/.config/vbas/vbas.zsh"
Enter
Type "PS1='$ '"
Enter
Type "clear"
Enter
Show

Sleep 600ms

# open folder dropdown
Type "cd "
Sleep 1800ms

# navigate to projects/
Down
Down
Sleep 400ms
Enter
Sleep 1200ms

# drill into myapp/
Down
Sleep 400ms
Enter
Sleep 1200ms

# drill into src/
Down
Sleep 400ms
Enter
Sleep 1000ms

# nothing deeper, escape
Escape
Sleep 600ms
