// define package
package main

// import packages
import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// parsepath ...
func parsepath(path string) string {

	homedir, err := os.UserHomeDir()

	if err != nil {
		panic(err)
	}
	
	if path == "~" {
		return homedir
	} else if strings.HasPrefix(path, "~") {
		return filepath.Join(homedir, path[2:])
	}

	// if "~" not at the start of path, the path doesn't need to be extended. Therefore, return it unchanged.
	return path
}

// parseconfigfile ...
func parseconfigfile(configdir string) map[string]string {

	var configs map[string]string
	configs = make(map[string]string)
	
	fileContent, err := os.ReadFile(configdir + "/.config/linuxmanager/config")
	
	if err != nil {
		panic(err)
	}

	fileContent_split := strings.Fields(string(fileContent))
	
	for _, s := range fileContent_split {
		row_split := strings.Split(s, "=")
		configs[row_split[0]] = row_split[1]
	} 

	return configs
}

// populateDropdown ...
func populateDropdown(mainapp *tview.Application, dropdown *tview.DropDown) {

	// use github api to get all public repositories of my user
	cmd, err := exec.Command("bash", "-c", "curl https://api.github.com/users/nomispaz/repos | grep full_name | cut -d':' -f 2 | cut -d'\"' -f 2").Output()
	if err != nil {
		panic(err)
	}
	// convert result byte to string and split at newline
	result := string(cmd)
	result_split := strings.Fields(result)

	// loop through slice and populate dropdown-menu
	for _, s := range result_split {
		dropdown.AddOption(s,nil)
	}
}

// populateMenu ...
func populateMenu(mainapp *tview.Application, menu *tview.List, contents *tview.TextView, footer *tview.TextView, curfolder string) {

	// remove all items from list
	for i := range menu.GetItemCount() {
		menu.RemoveItem(i)
	}

	// run ls command and parse it line by line to Menu list
	cmd := exec.Command("bash", "-c", "ls " + curfolder)

	// get a pipe to read from standard output
	stdout, _ := cmd.StdoutPipe()

	// Use the same pipe for standard error
	cmd.Stderr = cmd.Stdout

	// Make a new channel which will be used to ensure we get all output
	done := make(chan struct{})

	// Create a scanner which scans stdout in a line-by-line fashion
	cmd_scanner := bufio.NewScanner(stdout)

	// Use the scanner to scan the output line by line and log it
	// It's running in a goroutine so that it doesn't block
	go func() {
		// Read line by line and process it
		for cmd_scanner.Scan() {
			line := cmd_scanner.Text()
			curfile := curfolder + "/" + string(line)
			// AddItem(shortname, description, rune, function)
			menu.AddItem(string(line), "", '-',
				func() {
					// first check if entry is a dir or file
					info, err := os.Stat(curfile)
					if err != nil {
						fmt.Println("could not run command: ", err)
					}

					if !info.IsDir() {
						// entry is file --> display contents
						fileContent, err := os.ReadFile(curfile)
						if err != nil {
							panic(err)
						}
						contents.SetText(string(fileContent))
					} else {
						// folder was selected --> recreate list with entries in subfolder
						populateMenu(mainapp, menu, contents, footer, curfile)
						contents.SetText("")

					}
					footer.SetText(curfile)
				})
		}
		// We're all done, unblock the channel
		done <- struct{}{}
	}()

	// run command
	err := cmd.Start()

	if err != nil {
		panic(err)
	}

	// Wait for all output to be processed
	<-done

	// Wait for the command to finish
	err = cmd.Wait()

	// press b to go one level up
	menu.AddItem("Back", "One level up", 'b', func() {
		// cut string after last "/", i.e. cut last folder
		// --> resulting string contains folder one level up
		lastInd := strings.LastIndex(curfolder, "/")
		populateMenu(mainapp, menu, contents, footer, curfolder[:lastInd])
		contents.SetText("")
		footer.SetText(curfolder[:lastInd])
	})

	// press q to exit
	menu.AddItem("Quit", "Press to exit", 'q', func() {
		mainapp.Stop()
	})

}

func main() {

	app := tview.NewApplication()

	userhome, err := os.UserHomeDir()

	if err != nil {
		panic(err)
	}
	
	mConfig := parseconfigfile(userhome)

	startingfolder, keyexists := mConfig["defaultfolder"]

	startingfolder = parsepath(startingfolder)
	
	// if defaultfolder is not configured, use homedir as starting folder
	if !keyexists {
		startingfolder = userhome
	}
	
	// generate widgets
	wHeader := tview.NewTextView().SetTextAlign(tview.AlignCenter).SetText("Nomispaz linux manager")
	wFooter := tview.NewTextView().SetTextAlign(tview.AlignLeft).SetText(" ")
	wMenu := tview.NewList().ShowSecondaryText(false).SetMainTextColor(tcell.ColorNavy)
	wContents := tview.NewTextView().SetTextAlign(tview.AlignLeft).SetText(" ").SetDynamicColors(false).SetTextColor(tcell.ColorSlateGrey)
	wDropdown := tview.NewDropDown()

	wInputfield := tview.NewInputField().
		SetLabel("Destination: ").
		SetPlaceholder(startingfolder).
		SetDoneFunc(func(key tcell.Key) {
			app.Stop()
		})

	// set current folder to users home directory
	currentFolder := startingfolder

	grid := tview.NewGrid().
		SetRows(1, 1, 0, 1).
		SetColumns(40, 0).
		SetBorders(true).
		// p primitive, row, column, rowSpan, colSpan, minGridHeight, minGridWidth, focus bool
		AddItem(wHeader, 0, 0, 1, 2, 0, 0, false).
		AddItem(wDropdown, 1, 0, 1, 1, 0, 0, false).
		AddItem(wInputfield, 1, 1, 1, 1, 0, 0, false).
		AddItem(wFooter.SetText(currentFolder), 3, 0, 1, 2, 0, 0, false)

	grid.AddItem(wMenu, 2, 0, 1, 1, 0, 0, false).
		AddItem(wContents, 2, 1, 1, 1, 0, 0, false)

	populateDropdown(app, wDropdown)
	populateMenu(app, wMenu, wContents, wFooter, currentFolder)

	// check for keypress
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {

		// if key is ESC, stop app
		case tcell.KeyEsc:
			app.Stop()

		// change focus between Menu, Contents and Dropdown
		// Color of focused window is Green, of unfocused Grey
		case tcell.KeyTab:
			if wMenu.HasFocus() {
				app.SetFocus(wContents)
				wContents.SetTextColor(tcell.ColorNavy)
				wMenu.SetMainTextColor(tcell.ColorSlateGrey)
			} else if wContents.HasFocus() {
				app.SetFocus(wDropdown)
				wContents.SetTextColor(tcell.ColorSlateGrey)
				wMenu.SetMainTextColor(tcell.ColorSlateGray)
			} else if wDropdown.HasFocus() {
				app.SetFocus(wMenu)
				wContents.SetTextColor(tcell.ColorSlateGrey)
				wMenu.SetMainTextColor(tcell.ColorNavy)
			}
		case tcell.KeyBacktab:
			if wContents.HasFocus() {
				app.SetFocus(wMenu)
				wContents.SetTextColor(tcell.ColorSlateGrey)
				wMenu.SetMainTextColor(tcell.ColorNavy)
			} else if wMenu.HasFocus() {
				app.SetFocus(wDropdown)
				wContents.SetTextColor(tcell.ColorSlateGrey)
				wMenu.SetMainTextColor(tcell.ColorSlateGray)
			} else if wDropdown.HasFocus() {
				app.SetFocus(wContents)
				wContents.SetTextColor(tcell.ColorNavy)
				wMenu.SetMainTextColor(tcell.ColorSlateGray)
			}

		// if no special key is entered, check for "keyrunes", i.e. normal keys and numbers
		case tcell.KeyRune:
			switch event.Rune() {

			// a was entered
			case 'q':
				app.Stop()
			case 'e':
				exec.Command("bash", "-c", "chmod +x " + currentfile)
				exec.Command("bash", "-c", currentfile)
			}
		}
		return event
	})

	if err := app.SetRoot(grid, true).SetFocus(wMenu).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
