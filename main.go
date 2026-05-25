package main

/*
#include <locale.h>
*/
import "C"

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/gen2brain/go-mpv"
)

// Config mirrors the BSC TOML structure
type Config struct {
	Video VideoConfig `toml:"video"`
}

// VideoConfig holds the video-related configuration options from the TOML file
type VideoConfig struct {
	TargetDisplayName string `toml:"target_display_name"`
	Filepath          string `toml:"filepath"`
}

// displayStatus captures the results of the Wayland display validation
type displayStatus struct {
	isValid              bool
	availableMonitorsStr string
	isNonDefaultMonitor  bool
	foundMsg             string
	actualDisplay        string
	mpvTarget            string
}

func main() {

	var cfg Config
	if _, err := toml.DecodeFile("config.toml", &cfg); err != nil {

		log.Fatalf("########## Failed to parse TOML: %v", err)

	}

	log.Printf("########## Starting application with target display: '%s' and file: '%s'", cfg.Video.TargetDisplayName, cfg.Video.Filepath)

	app := adw.NewApplication("com.github.richbl.wayland-display-targeting", gio.ApplicationFlagsNone)
	app.ConnectActivate(func() { onActivate(app, &cfg) })

	if code := app.Run(os.Args); code > 0 {
		os.Exit(code)
	}

}

// onActivate sets up the GTK UI and initializes the mpv player based on the display validation results
func onActivate(app *adw.Application, cfg *Config) {

	log.Println("########## App Activated. Validating display name against Wayland compositor...")

	status := getDisplayStatus(cfg.Video.TargetDisplayName)
	player := initPlayer(status.mpvTarget, status.isNonDefaultMonitor)

	window := adw.NewApplicationWindow(&app.Application)
	window.SetTitle("Wayland Display Targeting Demo")
	window.SetDefaultSize(680, 320)

	topBox := gtk.NewBox(gtk.OrientationVertical, 0)
	topBox.Append(adw.NewHeaderBar())

	content := gtk.NewBox(gtk.OrientationVertical, 16)
	content.SetMarginTop(24)
	content.SetMarginBottom(24)
	content.SetMarginStart(32)
	content.SetMarginEnd(32)
	content.SetHExpand(true)
	content.SetVExpand(true)

	heading := gtk.NewLabel("")
	heading.SetMarkup("<span size='large' weight='bold'>Display Configuration</span>")
	heading.SetHAlign(gtk.AlignCenter)
	content.Append(heading)

	content.Append(buildInfoGrid(cfg, status))

	sep := gtk.NewSeparator(gtk.OrientationHorizontal)
	sep.SetMarginTop(4)
	sep.SetMarginBottom(4)
	content.Append(sep)

	content.Append(buildStartButton(window, player, cfg.Video.Filepath))

	topBox.Append(content)
	window.SetContent(topBox)

	window.ConnectCloseRequest(func() bool {

		log.Println("########## Shutting down mpv instance...")

		player.TerminateDestroy()

		return false
	})
	window.SetVisible(true)

}

// buildInfoGrid creates a GTK grid displaying the configuration and validation results
func buildInfoGrid(cfg *Config, status displayStatus) *gtk.Grid {

	grid := gtk.NewGrid()
	grid.SetRowSpacing(8)
	grid.SetColumnSpacing(20)
	grid.SetMarginTop(4)

	statusColor := "#e74c3c"
	if status.isValid {
		statusColor = "#2ecc71"
	}

	foundMarkup := fmt.Sprintf("<span foreground='%s' weight='bold'>%s</span>", statusColor, status.foundMsg)

	// Define the rows of information to display in the grid
	infoRows := []struct{ key, val, markup string }{
		{"Video filepath:", cfg.Video.Filepath, ""},
		{"Target monitor (via config.toml):", cfg.Video.TargetDisplayName, ""},
		{"Available physical monitor(s):", status.availableMonitorsStr, ""},
		{"Target monitor status:", "", foundMarkup},
		{"Playback monitor (mpv):", status.actualDisplay, ""},
	}

	// Populate the grid with labels based on the infoRows data
	for i, r := range infoRows {
		grid.Attach(makeKeyLabel(r.key), 0, i, 1, 1)
		if r.markup != "" {
			grid.Attach(makeMarkupLabel(r.markup), 1, i, 1, 1)
		} else {
			grid.Attach(makeValueLabel(r.val), 1, i, 1, 1)
		}
	}

	return grid
}

// buildStartButton creates the Start/Stop button and defines its click behavior to control mpv
// playback
func buildStartButton(window *adw.ApplicationWindow, player *mpv.Mpv, filepath string) *gtk.Button {

	btn := gtk.NewButtonWithLabel("Start")
	btn.SetSizeRequest(120, 40)
	btn.SetHAlign(gtk.AlignCenter)

	isPlaying := false

	btn.ConnectClicked(func() {
		if !isPlaying {

			log.Printf("########## Starting video: %s", filepath)

			if err := player.Command([]string{"loadfile", filepath}); err != nil {

				log.Printf("########## MPV Engine Error during loadfile: %v", err)

			} else {
				btn.SetLabel("Stop")
				isPlaying = true
			}
		} else {

			log.Println("########## Stop pressed. Closing application...")

			window.Close()
		}
	})

	return btn
}

// getDisplayStatus validates the target display name against the Wayland compositor and returns a
// structured status
func getDisplayStatus(targetName string) displayStatus {

	isValid, availableMonitors, matchedIndex := validateWaylandDisplay(targetName)

	availableMonitorsStr := strings.Join(availableMonitors, ", ")
	if len(availableMonitors) == 0 {
		availableMonitorsStr = "(none detected)"
	}

	status := displayStatus{
		isValid:              isValid,
		availableMonitorsStr: availableMonitorsStr,
		isNonDefaultMonitor:  isValid && matchedIndex > 0,
		mpvTarget:            targetName,
	}

	if isValid {
		status.foundMsg = fmt.Sprintf("Found: %s is available", targetName)
		status.actualDisplay = targetName

		if status.isNonDefaultMonitor {

			log.Printf("########## Success: Targeting validated display '%s' (non-default, index %d) — will use fullscreen.", targetName, matchedIndex)

		} else {

			log.Printf("########## Success: Targeting validated display '%s' (default monitor, index 0) — will use windowed mode.", targetName)

		}
	} else {
		status.foundMsg = fmt.Sprintf("Not found: %s (falling back to default)", targetName)
		fallbackMonitor := "Unknown"

		if len(availableMonitors) > 0 {
			fallbackMonitor = availableMonitors[0]
		}
		status.actualDisplay = fallbackMonitor + "(default)"
		status.mpvTarget = ""

		log.Printf("########## Warning: Target display '%s' not found. Falling back to %s.", targetName, fallbackMonitor)
	}

	return status
}

// makeKeyLabel creates a GTK label for the key/description column in the info grid
func makeKeyLabel(text string) *gtk.Label {

	lbl := gtk.NewLabel("")
	lbl.SetMarkup("<b>" + text + "</b>")
	lbl.SetHAlign(gtk.AlignStart)
	lbl.SetVAlign(gtk.AlignCenter)

	return lbl
}

// makeValueLabel creates a GTK label for the value column in the info grid
func makeValueLabel(text string) *gtk.Label {

	lbl := gtk.NewLabel(text)
	lbl.SetHAlign(gtk.AlignStart)
	lbl.SetVAlign(gtk.AlignCenter)
	lbl.SetSelectable(true)

	return lbl
}

// makeMarkupLabel creates a GTK label that can display markup (used for status messages in the
// info grid)
func makeMarkupLabel(markup string) *gtk.Label {

	lbl := gtk.NewLabel("")
	lbl.SetMarkup(markup)
	lbl.SetHAlign(gtk.AlignStart)
	lbl.SetVAlign(gtk.AlignCenter)

	return lbl
}

// validateWaylandDisplay checks if the requested connector is physically present
func validateWaylandDisplay(targetName string) (bool, []string, int) {

	disp := gdk.DisplayGetDefault()
	if disp == nil {

		log.Println("########## Error: Could not get default GDK display.")

		return false, nil, -1
	}

	monitors := disp.Monitors()
	total := int(monitors.NItems())

	log.Printf("########## GDK reports %d monitor(s) available.", total)

	found := false
	matchedIndex := -1
	var available []string

	// Iterate through monitors and log their details for debugging
	for i := range total {

		// Get the monitor item
		item := monitors.Item(uint(i))
		if item == nil {
			continue
		}

		mon, ok := item.Cast().(*gdk.Monitor)
		if !ok {

			log.Printf("########## Monitor %d: Could not cast to *gdk.Monitor", i)

			continue
		}

		connector := mon.Connector()
		model := mon.Model()
		manufacturer := mon.Manufacturer()

		log.Printf("########## Monitor %d details: Connector='%s', Model='%s', Manufacturer='%s'", i, connector, model, manufacturer)

		if connector != "" {
			available = append(available, connector)
		}

		if connector == targetName {

			log.Printf("########## --> Match found for target '%s' at index %d!", targetName, i)

			found = true
			matchedIndex = i
		}

	}

	return found, available, matchedIndex
}

// initPlayer sets up go-mpv and initializes it
func initPlayer(targetName string, isNonDefaultMonitor bool) *mpv.Mpv {

	// Force locale for libmpv safety
	C.setlocale(C.LC_NUMERIC, C.CString("C"))

	engine := mpv.New()

	// Set mpv option (Must be done BEFORE mpv initialization)
	setOpt := func(name, value string) {

		if err := engine.SetOptionString(name, value); err != nil {

			log.Printf("########## Error setting mpv option %s=%s: %v", name, value, err)

		} else {

			log.Printf("########## Set mpv option: %s=%s", name, value)

		}

	}

	setOpt("msg-level", "all=v")
	setOpt("keep-open", "yes")
	setOpt("gpu-context", "wayland")

	if targetName != "" && isNonDefaultMonitor {

		// Target found on a non-default monitor: use fullscreen since Wayland forces fullscreen for
		// non-default monitor windows
		setOpt("fs", "yes")
		setOpt("fs-screen-name", targetName)

		log.Printf("########## Targeting display '%s' (non-default) in fullscreen mode.", targetName)

	} else {

		// Either no target, or target is the default monitor (index 0): use a windowed size since
		// Wayland can manage default-monitor windows without forcing fullscreen
		setOpt("fs", "no")
		setOpt("autofit", "70%")

		if targetName != "" {

			log.Printf("########## Targeting display '%s' (default monitor, index 0) in windowed mode (70%%).", targetName)

		} else {

			log.Println("########## No target display name provided, mpv will use default screen (windowed).")

		}

	}

	// Initialize the mpv instance
	if err := engine.Initialize(); err != nil {

		log.Fatalf("########## Failed to initialize mpv instance: %v", err)

	}

	return engine
}
