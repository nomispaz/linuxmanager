// define package
package main

// import packages
import (
    "fmt"
    "os/exec"
    "os"
    "strings"
    "bufio"
    "github.com/rivo/tview"
    "github.com/gdamore/tcell/v2"
)

func createTextviewWidget(text string) tview.Primitive {
    return tview.NewTextView().
	    SetTextAlign(tview.AlignLeft).
            SetDynamicColors(true).
            SetText(text)
}

func populateMenu(mainapp *tview.Application, menu *tview.List, contents *tview.TextView, footer *tview.TextView, curfolder string) {

    // remove all items from list
    for i := range menu.GetItemCount() {
        menu.RemoveItem(i)
    }

    // run ls command and parse it line by line to Menu list
    cmd := exec.Command("ls", curfolder)
  
    // get a pipe to read from standard output
    stdout, _ := cmd.StdoutPipe()

    // Use the same pipe for standard error
    cmd.Stderr = cmd.Stdout

    // Make a new channel which will be used to ensure we get all output
    done := make(chan struct{})

    // Create a scanner which scans stdout in a line-by-line fashion
    scanner := bufio.NewScanner(stdout)

    // Use the scanner to scan the output line by line and log it
    // It's running in a goroutine so that it doesn't block
    go func() {
        // Read line by line and process it
        for scanner.Scan() {
            line := scanner.Text()
            curfile := curfolder + "/" + string(line)
            // AddItem(shortname, description, rune, function)
            menu.AddItem(string(line),"",'-',
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
                                fmt.Println("could not run command: ", err)
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
        fmt.Println("could not run command: ", err)
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

    // generate widgets
    wHeader   := tview.NewTextView().SetTextAlign(tview.AlignCenter).SetText("Nomispaz linux manager")
    wFooter   := tview.NewTextView().SetTextAlign(tview.AlignLeft).SetText(" ")
    wMenu := tview.NewList().ShowSecondaryText(false).SetMainTextColor(tcell.ColorNavy)
    wContents := tview.NewTextView().SetTextAlign(tview.AlignLeft).SetText(" ").SetDynamicColors(false).SetTextColor(tcell.ColorSlateGrey)

    currentFolder := "/home/simonheise/git_repos/ArchInstall"
    
    grid := tview.NewGrid().
        SetRows(1, 0, 1).
	SetColumns(40, 0).
	SetBorders(true).
// p primitive, row, column, rowSpan, colSpan, minGridHeight, minGridWidth, focus bool
	AddItem(wHeader, 0, 0, 1, 2, 0, 0, false).
        AddItem(wFooter.SetText(currentFolder), 2, 0, 1, 2, 0, 0, false)

    // Layout for screens narrower than 100 cells (menu bar is hidden).
    // grid.AddItem(menu, 0, 0, 0, 0, 0, 0, false).
    //     AddItem(main, 1, 0, 1, 2, 0, 0, false)
    // Layout for screens wider than 100 cells.
    // grid.AddItem(menu, 1, 0, 1, 1, 0, 100, false).
    //     AddItem(main, 1, 1, 1, 1, 0, 100, false)

    grid.AddItem(wMenu, 1, 0, 1, 1, 0, 0, false).
         AddItem(wContents, 1, 1, 1, 1, 0, 0, false)

    
    populateMenu(app, wMenu, wContents, wFooter, currentFolder)
  
    // check for keypress
    app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
        switch event.Key() {

        // if key is ESC, stop app
        case tcell.KeyEsc:
            app.Stop()

        // if key is TAB, change focus between Menu and Contents
        // Color of focused window is Green, of unfocused Grey
        case tcell.KeyTab:
            if wMenu.HasFocus() {
                app.SetFocus(wContents)
                wContents.SetTextColor(tcell.ColorNavy)
                wMenu.SetMainTextColor(tcell.ColorSlateGrey)
            } else {
                app.SetFocus(wMenu)
                wContents.SetTextColor(tcell.ColorSlateGrey)
                wMenu.SetMainTextColor(tcell.ColorNavy)

            }

        // if no special key is entered, check for "keyrunes", i.e. normal keys and numbers
        case tcell.KeyRune:
            switch event.Rune() {

            // a was entered
            case 'q':
                app.Stop()
            }
        }    
	return event
    })

    if err := app.SetRoot(grid, true).SetFocus(wMenu).EnableMouse(true).Run(); err !=nil {
        panic(err)
    }
}
