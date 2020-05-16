# octopimon
TUI for OctoPi


### .bashrc
Needed a term that supports weird unicode and stuff so using framebuffer to start a terminal with a font that supports the symbols used by asciigraph and such.
```bash
# Only run this on the local TTY
if [[ "$(tty)" == "/dev/tty1" ]]
then
  FRAMEBUFFER=/dev/fb1 fbterm -s 10 -n Monaco OP_TOKEN="<token>" /home/pi/octopimon
fi
```