package main

import (
	_ "embed"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
)

//go:embed icons/icon.png
var iconBytes []byte

func main() {

	if err := initTools(); err != nil {
		fmt.Printf("Failed to initialize tools: %v\n", err)
		return
	}

	defer cleanupTools()

	a := app.NewWithID("iRay")

	icon := fyne.NewStaticResource("icon", iconBytes)
	a.SetIcon(icon)

	w := a.NewWindow("Media Controller")
	w.Resize(fyne.NewSize(680, 520))

	tabs := container.NewAppTabs(
		container.NewTabItem("Video", buildVideoUI(w)),
		container.NewTabItem("Photo", buildPhotoUI(w)),
		container.NewTabItem("YouTube", buildYoutubeUI(w)),
	)
	tabs.SetTabLocation(container.TabLocationTop)

	w.SetContent(tabs)
	w.ShowAndRun()
}
