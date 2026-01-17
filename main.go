package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"

	"bitbucket.org/rj/goey"
	"bitbucket.org/rj/goey/base"
	"bitbucket.org/rj/goey/loop"
	"bitbucket.org/rj/goey/windows"
)




var(
	wMainWin *windows.Window
	wChannel = "release"
	wUsername = "TransRights"
	wVersion = 1
	wLatestVersions = map[string]int{}
	wProgress = 0
	wDisabled = false
)


func createWindow() error {
	w, err := windows.NewWindow("Trans Rights", renderWindow())
	if err != nil {
		return err
	}

	w.SetScroll(false, false);

	wMainWin = w;

	return nil
}


func valToChannel(vchl int) string {
	if vchl == 0 {
		return "release";
	} else if vchl == 1 {
		return "pre-release"
	}
	return "release";
}

func channelToVal(channel string) int {
	if channel == "release"{
		return 0;
	} else if channel == "pre-release" {
		return 1;
	}
	return 0;
}

func patchLineMenu() base.Widget {
	return &goey.VBox{
		AlignMain: goey.SpaceBetween,
		Children: []base.Widget{
			&goey.Label{Text: "Patchline:"},
			&goey.SelectInput{
				Items: []string {
					"release",
					"pre-release",
				},
				Value: channelToVal(wChannel),
				Disabled: wDisabled,
				OnChange: func(v int) {
					if wChannel != valToChannel(v) {
						wChannel = valToChannel(v);
						updateWindow();
					}
				},
			},
		},
	};
}




func versionMenu() base.Widget {
	versions := goey.SelectInput {
		OnChange: func(v int) { wVersion = v+1},
		Disabled: wDisabled,
	};

	for i := range wLatestVersions[wChannel] {
		versions.Items = append(versions.Items, "Version "+strconv.Itoa(i+1)+ "");
	}

	versions.Value = wLatestVersions[wChannel];

	return &goey.VBox{
		AlignMain: goey.SpaceBetween,
		Children: []base.Widget{
			&goey.Label{Text: "Version:"},
			&versions,
		},
	};
}

func usernameBox() base.Widget {
	return &goey.VBox{
		AlignMain: goey.SpaceBetween,
		Children: []base.Widget{
			&goey.Label{Text: "Username:"},
			&goey.TextInput{
					Value: wUsername,
					Placeholder: "Username",
					Disabled: wDisabled,
					OnChange: func(v string) {
						wUsername = v;
					},
			},
		},
	};
}

func updateWindow() {
	err := wMainWin.SetChild(renderWindow())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
	}
}

func updateProgress(done int64, total int64) {
	lastProgress := wProgress;
	wProgress = int((float64(done) / float64(total)) * 100.0);
	if lastProgress != wProgress{
		updateWindow();
	}
}

func renderWindow() base.Widget {
	return &goey.Padding{
		Insets: goey.DefaultInsets(),
		Child: &goey.Align{
			Child: &goey.VBox{
				AlignMain: goey.MainStart,
				Children: []base.Widget{

					usernameBox(),
					patchLineMenu(),
					versionMenu(),

					&goey.Progress{
						Value: wProgress,
						Min: 0,
						Max: 100,
					},
					&goey.Button{
						Text: "Start Game",
						Disabled: wDisabled,
						OnClick: func() {
							go func() {
								wDisabled = true;
								installJre(updateProgress);
								installGame(wVersion, wChannel, updateProgress);
								launchGame(wVersion, wChannel, wUsername, usernameToUuid(wUsername));
								wDisabled = false;
							}();
						},
					},
				},
			},
		},
	}
}


func main() {

	os.MkdirAll(MAIN_FOLDER, 0666);
	os.MkdirAll(GAME_FOLDER, 0666);
	os.MkdirAll(USERDATA_FOLDER, 0666);
	os.MkdirAll(JRE_FOLDER, 0666);

	wLatestVersions["release"] = findLatestVersionNoAuth(runtime.GOARCH, runtime.GOOS, "release");
	wLatestVersions["pre-release"] = findLatestVersionNoAuth(runtime.GOARCH, runtime.GOOS, "pre-release");

	err := loop.Run(createWindow)
	if err != nil {
		fmt.Println("Error: ", err)
	}

}
