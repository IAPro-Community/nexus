#!/usr/bin/env python3
"""Small XTest helper for the real Wails window under Xvfb."""

from __future__ import annotations

import ctypes
import sys
import time

x11 = ctypes.CDLL("libX11.so.6")
xtst = ctypes.CDLL("libXtst.so.6")
x11.XOpenDisplay.argtypes = [ctypes.c_char_p]
x11.XOpenDisplay.restype = ctypes.c_void_p
x11.XCloseDisplay.argtypes = [ctypes.c_void_p]
x11.XSetInputFocus.argtypes = [ctypes.c_void_p, ctypes.c_ulong, ctypes.c_int, ctypes.c_ulong]
x11.XKeysymToKeycode.argtypes = [ctypes.c_void_p, ctypes.c_ulong]
x11.XKeysymToKeycode.restype = ctypes.c_uint
x11.XSync.argtypes = [ctypes.c_void_p, ctypes.c_int]
xtst.XTestFakeMotionEvent.argtypes = [
    ctypes.c_void_p,
    ctypes.c_int,
    ctypes.c_int,
    ctypes.c_int,
    ctypes.c_ulong,
]
xtst.XTestFakeButtonEvent.argtypes = [
    ctypes.c_void_p,
    ctypes.c_uint,
    ctypes.c_int,
    ctypes.c_ulong,
]
xtst.XTestFakeKeyEvent.argtypes = [
    ctypes.c_void_p,
    ctypes.c_uint,
    ctypes.c_int,
    ctypes.c_ulong,
]

display = x11.XOpenDisplay(None)
if not display:
    raise SystemExit("XOpenDisplay failed")


def keycode(keysym: int) -> int:
    value = x11.XKeysymToKeycode(display, keysym)
    if not value:
        raise ValueError(f"no X11 keycode for keysym {keysym}")
    return value


def click(x: int, y: int) -> None:
    xtst.XTestFakeMotionEvent(display, 0, x, y, 0)
    xtst.XTestFakeButtonEvent(display, 1, 1, 0)
    xtst.XTestFakeButtonEvent(display, 1, 0, 0)
    x11.XSync(display, 0)
    time.sleep(0.3)


def move(x: int, y: int) -> None:
    xtst.XTestFakeMotionEvent(display, 0, x, y, 0)
    x11.XSync(display, 0)


def type_text(value: str) -> None:
    for char in value:
        if char.isupper():
            shift = keycode(0xFFE1)
            code = keycode(ord(char.lower()))
            xtst.XTestFakeKeyEvent(display, shift, 1, 0)
            xtst.XTestFakeKeyEvent(display, code, 1, 0)
            xtst.XTestFakeKeyEvent(display, code, 0, 0)
            xtst.XTestFakeKeyEvent(display, shift, 0, 0)
        else:
            code = keycode(ord(char))
            xtst.XTestFakeKeyEvent(display, code, 1, 0)
            xtst.XTestFakeKeyEvent(display, code, 0, 0)
        x11.XSync(display, 0)
        time.sleep(0.02)


def press(keysym: int) -> None:
    code = keycode(keysym)
    xtst.XTestFakeKeyEvent(display, code, 1, 0)
    xtst.XTestFakeKeyEvent(display, code, 0, 0)
    x11.XSync(display, 0)
    time.sleep(0.025)


def chord(modifier: int, char: str) -> None:
    mod = keycode(modifier)
    code = keycode(ord(char.lower()))
    xtst.XTestFakeKeyEvent(display, mod, 1, 0)
    xtst.XTestFakeKeyEvent(display, code, 1, 0)
    xtst.XTestFakeKeyEvent(display, code, 0, 0)
    xtst.XTestFakeKeyEvent(display, mod, 0, 0)
    x11.XSync(display, 0)
    time.sleep(0.3)


args = sys.argv[1:]
if args[0] == "click" and len(args) == 3:
    click(int(args[1]), int(args[2]))
elif args[0] == "move" and len(args) == 3:
    move(int(args[1]), int(args[2]))
elif args[0] == "type" and len(args) == 2:
    type_text(args[1])
elif args[0] == "ctrl" and len(args) == 2:
    chord(0xFFE3, args[1])
elif args[0] == "key" and len(args) == 2:
    keys = {
        "ENTER": 0xFF0D,
        "ESC": 0xFF1B,
        "TAB": 0xFF09,
        "SPACE": 0x20,
        "LEFT": 0xFF51,
        "UP": 0xFF52,
        "RIGHT": 0xFF53,
        "DOWN": 0xFF54,
    }
    press(keys[args[1]])
elif args[0] == "focus" and len(args) == 2:
    x11.XSetInputFocus(display, int(args[1], 0), 2, 0)
    x11.XSync(display, 0)
else:
    raise SystemExit("usage: x11_input.py click x y | move x y | type text | ctrl key | key NAME | focus WINDOW_ID")

x11.XCloseDisplay(display)
