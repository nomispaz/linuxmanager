// define package
package main

// import packages
import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// displayHelp ...
func (t *Tui) displayHelp()  {
	var helptext string
	helptext = "Help"
	helptext += "\nc: Clone selected repo into destination"
	helptext += "\ne: Execute selected script"
	helptext += "\nESC: Quit program"
	helptext += "\nF1: Open Help"
	helptext += "\nF2: Switch to input field"
	t.contents.SetText(helptext)
}

// rotate through widgets of grid according to nexItem (negative means backwords)
func (t *Tui) focusGridItem(inkrement int)  {
	
}

// parseCmdOutput: parses up to 100 different commands and feeds result to contents-widget
func (t *Tui) parseCmdOutput(cmdArray [100]string)  {

	var output string
	
	for idx := range len(cmdArray) {
		if cmdArray[idx] != "" {
			
			cmd := exec.Command("bash", "-c", cmdArray[idx])

			output += "Command: " + cmdArray[idx] + "\n"

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
					output += line + "\n"
					t.contents.SetText(output)
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
			
		} else {
			break
		}
	}
	
		
}

// parsepath: resolve "~" in path to home directory
func parsepath(path string, userhome string) string {

	if path == "~" {
		return userhome
	} else if strings.HasPrefix(path, "~") {
		return filepath.Join(userhome, path[2:])
	}

	// if "~" not at the start of path, the path doesn't need to be resolved. Therefore, return it unchanged.
	return path
}

// parseconfigfile: read config file from users configdir and parse settings
func parseconfigfile() map[string]string {

	configs := make(map[string]string)

	configdir, err := os.UserConfigDir()

	if err != nil {
		panic(err)
	}
	
	fileContent, err := os.ReadFile(configdir + "/linuxmanager/config")
	
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

// create structure for terminal interface
type Tui struct {
	app        *tview.Application
	grid       *tview.Grid
	inputfield *tview.InputField
	header     *tview.TextView
	menu       *tview.List
	contents   *tview.TextView
	footer     *tview.TextView
	dropdown   *tview.DropDown

	selectedMenuEntry string
	selectedMenuItemIdx int
	curfolder string

	mConfig map[string]string

}

// function to create basic app with strct Tui
func CreateApplication() *Tui {
	return new(Tui)
}

// initialize the Tui struct
func (t *Tui) Init() {
	t.app = tview.NewApplication()
	t.grid = tview.NewGrid()
	t.inputfield = tview.NewInputField()
	t.header = tview.NewTextView()
	t.menu = tview.NewList()
	t.contents = tview.NewTextView()
	t.footer = tview.NewTextView()
	t.dropdown = tview.NewDropDown()

	t.selectedMenuEntry = ""
	t.selectedMenuItemIdx = -1
	t.curfolder = ""
}

// setup the initial TUI
func (t *Tui) SetupTUI()  {

 	// generate widgets
	t.header.SetTextAlign(tview.AlignCenter).SetText("Nomispaz linux manager")
	t.footer.SetTextAlign(tview.AlignLeft).SetText(" ")
	t.menu.ShowSecondaryText(false).SetMainTextColor(tcell.ColorNavy)
	t.contents.SetTextAlign(tview.AlignLeft).SetText(" ").SetDynamicColors(false).SetTextColor(tcell.ColorSlateGrey)

	t.inputfield.SetLabel("Select folder: ").
		SetDoneFunc(func(key tcell.Key) {
			// first check if entry is a dir or file
			log.Println(t.inputfield.GetText())
			info, err := os.Stat(t.inputfield.GetText())
			if err != nil {
				log.Println("Cannot stat " + t.inputfield.GetText())
			}

			if info.IsDir() {
				t.curfolder = t.inputfield.GetText()
				t.populateMenu()
			} else {
				t.contents.SetText("No valid folder entered")
			}
		})

	t.grid.SetRows(1, 1, 1, 0, 1).
		SetColumns(40, 0).
		SetBorders(true).
		// p primitive, row, column, rowSpan, colSpan, minGridHeight, minGridWidth, focus bool
		AddItem(t.header, 0, 0, 1, 2, 0, 0, false).
		AddItem(t.dropdown, 1, 0, 1, 1, 0, 0, false).
		AddItem(t.inputfield, 2, 0, 1, 1, 0, 0, false).
		AddItem(t.menu, 3, 0, 1, 1, 0, 0, false).
		AddItem(t.contents, 1, 1, 3, 1, 0, 0, false).
		AddItem(t.footer.SetText(t.curfolder), 4, 0, 1, 2, 0, 0, false)

}

// populateDropdown: populate the dropdown menu
func (t *Tui) populateDropdown(userhome string) {

	t.dropdown.AddOption("Git repos online", func() {
		configoption, keyexist := t.mConfig["gitfolder"]
		if keyexist {
			t.curfolder = parsepath(configoption, userhome)
		}
		t.populateMenuGit()
		t.app.SetFocus(t.menu)
		t.menu.SetMainTextColor(tcell.ColorNavy)
		t.inputfield.SetText(t.curfolder)
			})
	t.dropdown.AddOption("Git repos offline", func() {
		configoption, keyexist := t.mConfig["gitfolder"]
		if keyexist {
			t.curfolder = parsepath(configoption, userhome)
		}
		t.populateMenu()
		t.app.SetFocus(t.menu)
		t.menu.SetMainTextColor(tcell.ColorNavy)
		t.inputfield.SetText(t.curfolder)

	})
	t.dropdown.AddOption("File browser", nil)
}

//  populate menu when online git repo was selected
func (t *Tui) populateMenuGit()  {
	
	// set the function that is run when a menu item was selected
	t.menu.SetSelectedFunc(func(i int, main string, secondary string, shortcut rune) {
		t.selectedMenuItemIdx = t.menu.GetCurrentItem()
		var secondarytext string
		t.selectedMenuEntry, secondarytext = t.menu.GetItemText(t.selectedMenuItemIdx)
		if secondarytext != "" {
			t.selectedMenuEntry += ", " + secondarytext
		}
		if t.selectedMenuItemIdx == 0 {
			t.contents.SetText("To clone the repository, enter c.")
		}
		t.app.SetFocus(t.contents)
		t.contents.SetTextColor(tcell.ColorNavy)
		t.menu.SetMainTextColor(tcell.ColorSlateGrey)
	})
	
	// remove all items from list
	for i := range t.menu.GetItemCount() {
		t.menu.RemoveItem(i)
	}

	// use github api to get all public repositories of my user
	cmd, err := exec.Command("bash", "-c", "curl https://api.github.com/users/nomispaz/repos | grep full_name | cut -d':' -f 2 | cut -d'\"' -f 2").Output()
	if err != nil {
		panic(err)
	}
	// convert result byte to string and split at newline
	result := string(cmd)
	result_split := strings.Fields(result)

	for _, s := range result_split {
		t.menu.AddItem(strings.Split(s, "/")[1], "", '-', nil)
	}
}

// populateMenu populate the side menu
func (t *Tui) populateMenu() {
	
	// set the function that is run when a menu item was selected
	t.menu.SetSelectedFunc(func(i int, main string, secondary string, shortcut rune) {
		if !(shortcut=='q' || shortcut=='b') {
			t.selectedMenuEntry = t.curfolder + "/" + main

			// first check if entry is a dir or file
			info, err := os.Stat(t.selectedMenuEntry)
			if err != nil {
				fmt.Println("could not run command: ", err)
			}

			if !info.IsDir() {
				// entry is file --> display contents
				fileContent, err := os.ReadFile(t.selectedMenuEntry)
				if err != nil {
					panic(err)
				}
				t.contents.SetText(string(fileContent))
				t.app.SetFocus(t.contents)
				t.contents.SetTextColor(tcell.ColorNavy)
				t.menu.SetMainTextColor(tcell.ColorSlateGrey)					
			} else {
				// folder was selected --> recreate list with entries in subfolder
				t.curfolder = t.selectedMenuEntry
				t.contents.SetText("")
				t.populateMenu()
			}
			t.footer.SetText(t.selectedMenuEntry)			
		}
	})

	
	// remove all items from list
	for i := range t.menu.GetItemCount() {
		t.menu.RemoveItem(i)
	}

	// run ls command and parse it line by line to Menu list
	cmd := exec.Command("bash", "-c", "ls " + t.curfolder)

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
			t.selectedMenuEntry = t.curfolder + "/" + string(line)
			// AddItem(shortname, description, rune, function)
			t.menu.AddItem(string(line), "", '-',nil)
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
	t.menu.AddItem("Back", "One level up", 'b', func() {
		// cut string after last "/", i.e. cut last folder
		// --> resulting string contains folder one level up
		lastInd := strings.LastIndex(t.curfolder, "/")
		t.curfolder = t.curfolder[:lastInd]
		t.contents.SetText("")
		t.footer.SetText(t.curfolder[:lastInd])
		t.populateMenu()
	})

	// press q to exit
	t.menu.AddItem("Quit", "Press to exit", 'q', func() {
		t.app.Stop()
	})

}

// Keybindungs ...
func (t *Tui) Keybindings()  {
	
	t.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {

		// if key is ESC, stop app
		case tcell.KeyEsc:
			t.app.Stop()
		case tcell.KeyF1:
			t.displayHelp()
		case tcell.KeyF2:
			t.app.SetFocus(t.inputfield)
			t.contents.SetTextColor(tcell.ColorSlateGrey)
			t.menu.SetMainTextColor(tcell.ColorSlateGrey)

		// change focus between Menu, Contents and Dropdown
		// Color of focused window is Green, of unfocused Grey
		
		case tcell.KeyTab:
			if t.menu.HasFocus() {
				t.app.SetFocus(t.contents)
				t.contents.SetTextColor(tcell.ColorNavy)
				t.menu.SetMainTextColor(tcell.ColorSlateGrey)
			} else if t.contents.HasFocus() {
				t.app.SetFocus(t.dropdown)
				t.contents.SetTextColor(tcell.ColorSlateGrey)
				t.menu.SetMainTextColor(tcell.ColorSlateGray)
			} else if t.dropdown.HasFocus() {
				t.app.SetFocus(t.menu)
				t.contents.SetTextColor(tcell.ColorSlateGrey)
				t.menu.SetMainTextColor(tcell.ColorNavy)
			} else if t.inputfield.HasFocus() {
				t.app.SetFocus(t.menu)
				t.contents.SetTextColor(tcell.ColorSlateGrey)
				t.menu.SetMainTextColor(tcell.ColorNavy)
			}
		case tcell.KeyBacktab:
			if t.contents.HasFocus() {
				t.app.SetFocus(t.menu)
				t.contents.SetTextColor(tcell.ColorSlateGrey)
				t.menu.SetMainTextColor(tcell.ColorNavy)
			} else if t.menu.HasFocus() {
				t.app.SetFocus(t.dropdown)
				t.contents.SetTextColor(tcell.ColorSlateGrey)
				t.menu.SetMainTextColor(tcell.ColorSlateGray)
			} else if t.dropdown.HasFocus() {
				t.app.SetFocus(t.contents)
				t.contents.SetTextColor(tcell.ColorNavy)
				t.menu.SetMainTextColor(tcell.ColorSlateGray)
			}

		// if no special key is entered, check for "keyrunes", i.e. normal keys and numbers
		case tcell.KeyRune:
			switch event.Rune() {
			// execute selected script
			// only if file was selected
			case 'e':
				if t.contents.HasFocus() {
					info, err := os.Stat(t.selectedMenuEntry)
					if err != nil {
						panic(err)
					}
					if !info.IsDir() {
						var cmdArray [100]string
						cmdArray[0] = "chmod +x " + t.selectedMenuEntry
						cmdArray[1] = t.selectedMenuEntry
						t.parseCmdOutput(cmdArray)
					}
				}
			case 'c':
				if t.contents.HasFocus() {
					selectionidx, dropdownselection := t.dropdown.GetCurrentOption()
					configoption, keyexist := t.mConfig["gituser"]
					if !keyexist {
						t.contents.SetText("No gituser specified in configs")
					}
					if selectionidx == 0 && dropdownselection != "" {
						var cmdArray [100]string
						cmdArray[0] = "git clone https://github.com/" + configoption + "/" + t.selectedMenuEntry + " " + t.curfolder + "/" + t.selectedMenuEntry
						t.parseCmdOutput(cmdArray)
					}
				}
			case 'p':
				if t.contents.HasFocus() {
					selectionidx, dropdownselection := t.dropdown.GetCurrentOption()
					configoption, keyexist := t.mConfig["gituser"]
					if !keyexist {
						t.contents.SetText("No gituser specified in configs")
					}
					t.contents.SetText("Modus " + dropdownselection + " Pushing selected repository " + t.curfolder + "/" + t.selectedMenuEntry + " to git user nomispaz" + configoption + dropdownselection + string(selectionidx))
					//		if selectionidx == 1 {
					//	cmd := exec.Command("bash", "-c",
					//	"cd " + )
					//}
				}
			}
		}
		return event
	})
}

// main ...
func main()  {
	tui := CreateApplication()
	tui.Init()
	tui.SetupTUI()
	tui.Keybindings()

	// setup initial folder (read from configfile or user home as standard)
	userhome, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	mConfig := parseconfigfile()

	tui.mConfig = mConfig

	startingfolder, keyexists := mConfig["defaultfolder"]

	// if defaultfolder is not configured, use homedir as starting folder
	if !keyexists {
		startingfolder = userhome
	} else {
		startingfolder = parsepath(startingfolder, userhome)
	}

	tui.curfolder = startingfolder
	
	// fill initialized Tui with entries
	tui.inputfield.SetPlaceholder(startingfolder)
	tui.populateDropdown(userhome)
	tui.populateMenu()
	tui.displayHelp()
	
	if err := tui.app.SetRoot(tui.grid, true).SetFocus(tui.menu).EnableMouse(true).Run(); err != nil {
		panic(err)
	}

}
