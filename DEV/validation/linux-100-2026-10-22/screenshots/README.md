# Screenshot evidence

No screenshots are included because this GNOME 49 compositor denied available capture methods during the native run. `grim` reported unsupported `wlr-screencopy-unstable-v1`; GNOME Shell returned `AccessDenied: Screenshot is not allowed`; Flameshot timed out. The raw command results are in `../logs/native-visual-capture.log`.

The live Wails window and page were confirmed through the Wayland session's AT-SPI tree in `../logs/native-accessibility-session-bus.log`. AT-SPI text is not a screenshot and does not certify colors, transparency, clipping, theme rendering, animation, or visual layout.

