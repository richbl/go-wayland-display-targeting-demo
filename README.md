## Overview

**Go Wayland Display Targeting Demo** is a simple Go-based demo that identifies available Wayland monitors for video playback of the [mpv](https://mpv.io/) media player.

<p align="center">
<img width="850" alt="Screenshot showing WDTD GUI" src="https://raw.githubusercontent.com/richbl/go-wayland-display-targeting-demo/refs/heads/main/.github/assets/demo_monitor_success.png">
</p>

<p align="center">
<img width="850" alt="Screenshot showing WDTD GUI" src="https://raw.githubusercontent.com/richbl/go-wayland-display-targeting-demo/refs/heads/main/.github/assets/demo_monitor_failure.png">
</p>

It's presented here as a simple project demo to serve as a functional example of how to implement a Wayland display targeting protocol in Go.

For an example of how this display targeting logic is used in a more significant project, check out the [**BLE Sync Cycle project**](https://github.com/richbl/go-ble-sync-cycle).

## Installation

A note on installation: since the GTK4/Adwaita packages ([GoTK4](https://github.com/diamondburned/gotk4)/[GoTK4-Adwaita](https://github.com/diamondburned/gotk4-adwaita)) require local compilation (the native libraries are written in C), expect a delay in first-time application execution (~10-15 minutes depending on CPU speed).
