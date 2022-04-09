package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/awesome-gocui/gocui"
	"github.com/mmcdole/gofeed"
)

type Feeds struct {
	Feed string
}

var (
	feeds                        []Feeds
	addFeedViewTop               bool = false
	curFeed                           = 0
	curTitle                          = 0
	numberOfTitlesInSelectedFeed      = 0
)

func greyHighlightText(text string) string {
	return "\x1b[7;7m" + text + "\033[0m"
}

func getFeedTitle(rss string) string {
	fp := gofeed.NewParser()
	fp.UserAgent = "MyCustomAgent 1.0"
	feed, _ := fp.ParseURL(rss)
	return feed.Title
}

func getFeedTitles(selectedIndex int) string {

	text := ""

	for i := 0; i < len(feeds); i++ {
		if i == selectedIndex {
			text += greyHighlightText(getFeedTitle(feeds[i].Feed)) + "\n"
		} else {
			text += getFeedTitle(feeds[i].Feed) + "\n"
		}

	}
	return text
}

func getFeedItemTitles(selectedFeedIndex int, selectedTitleIndex int) string {
	text := ""
	if len(feeds) > 0 {
		fp := gofeed.NewParser()
		fp.UserAgent = "MyCustomAgent 1.0"
		feed, _ := fp.ParseURL(feeds[selectedFeedIndex].Feed)
		numberOfTitlesInSelectedFeed = len(feed.Items)
		for i := 0; i < len(feed.Items); i++ {
			if i == selectedTitleIndex {
				text += greyHighlightText(feed.Items[i].Title) + "\t\t" + "\n"
			} else {
				text += feed.Items[i].Title + "\t\t" + "\n"
			}

		}
	}

	return text
}

func overwrite(g *gocui.Gui, v *gocui.View) error {
	v.Overwrite = !v.Overwrite
	return nil
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	if v, err := g.SetView("statusbar", 0, 0, maxX-1, 2, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		fmt.Fprintln(v, "\x1b[7;7mTerminalRSS")
	}
	if v, err := g.SetView("cmdline", 0, maxY-5, maxX-1, maxY-1, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		fmt.Fprintln(v, "\x1b[7;7m^X\033[0m Exit\t\x1b[7;7m^A/^D\033[0m Change Feed\t\x1b[7;7m^W/^S\033[0m Change Item\t\x1b[7;7m^Space Bar\033[0m Open Item\t\x1b[7;7m^E\033[0m Add Feed\t")
	}
	if v, err := g.SetView("feedlist", 0, 3, maxX/4, maxY-6, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Wrap = true
		fmt.Fprintln(v, getFeedTitles(curFeed))
	}

	if v, err := g.SetView("addFeeds", maxX/4+1, 3, maxX-1, maxY-6, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Wrap = true
		v.Title = "Enter RSS URLs, each line is one URL."
		v.Editable = true
	}

	if v, err := g.SetView("main", maxX/4+1, 3, maxX-1, maxY-6, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Wrap = true

		if _, err := g.SetCurrentView("main"); err != nil {
			return err
		}
		if len(feeds) > 0 {
			fmt.Fprintln(v, getFeedItemTitles(curFeed, 0))
		}

	}

	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func showAddFeedInput(g *gocui.Gui, v *gocui.View) error {
	g.SetViewOnTop("addFeeds")
	g.Cursor = true
	if _, err := g.SetCurrentView("addFeeds"); err != nil {
		return err
	}
	g.Update(func(g *gocui.Gui) error {
		v, err := g.View("cmdline")
		if err != nil {
			return err
		}
		v.Clear()
		fmt.Fprintln(v, "\x1b[7;7m^X\033[0m Exit\t\x1b[7;7m^E\033[0m Add Feed\t\x1b[7;7m^R\033[0m Save//Quit")
		return nil
	})

	return nil
}

func isURL(line string) bool {
	regexExpression := regexp.MustCompile(`https?://(www.)?[-a-zA-Z0-9@:%._+~#=]{1,256}.[a-zA-Z0-9()]{1,6}/?`)
	if regexExpression.FindStringSubmatch(line) != nil {
		return true
	}
	return false
}

func handleAddFeedInput(g *gocui.Gui, v *gocui.View) error {
	f, err := g.View("addFeeds")
	if err != nil {
		return err
	}
	var lines = f.BufferLines()
	f.Clear()
	for i := 0; i < len(lines); i++ {
		if isURL(strings.TrimSpace(lines[i])) {
			newStruct := &Feeds{
				Feed: lines[i],
			}
			feeds = append(feeds, *newStruct)
		}
	}

	dataBytes, err := json.Marshal(feeds)
	if err != nil {
		fmt.Println(err)
	}

	err = ioutil.WriteFile("config.json", dataBytes, 0644)
	if err != nil {
		fmt.Println(err)
	}

	g.SetViewOnTop("main")
	g.Cursor = false
	if _, err := g.SetCurrentView("main"); err != nil {
		return err
	}

	g.Update(func(g *gocui.Gui) error {
		v, err := g.View("cmdline")
		if err != nil {
			return err
		}
		v.Clear()
		fmt.Fprintln(v, "\x1b[7;7m^X\033[0m Exit\t\x1b[7;7m^A/^D\033[0m Change Feed\t\x1b[7;7m^W/^S\033[0m Change Item\t\x1b[7;7m^Space Bar\033[0m Open Item\t\x1b[7;7m^E\033[0m Add Feed\t")
		return nil
	})

	if len(feeds) > 0 {
		g.Update(func(g *gocui.Gui) error {
			v, err := g.View("feedlist")
			if err != nil {
				return err
			}
			v.Clear()
			fmt.Fprintln(v, getFeedTitles(0))
			return nil
		})

		g.Update(func(g *gocui.Gui) error {
			v, err := g.View("main")
			if err != nil {
				return err
			}
			v.Clear()
			fmt.Fprintln(v, getFeedItemTitles(0, 0))
			return nil
		})
	}

	return nil
}

func nextFeedDown(g *gocui.Gui, disableCurrent bool) error {
	next := curFeed + 1
	if next > len(feeds)-1 {
		next = 0
	}

	g.Update(func(g *gocui.Gui) error {
		v, err := g.View("feedlist")
		if err != nil {
			return err
		}
		v.Clear()
		fmt.Fprintln(v, getFeedTitles(next))
		return nil
	})

	g.Update(func(g *gocui.Gui) error {
		v, err := g.View("main")
		if err != nil {
			return err
		}
		v.Clear()
		fmt.Fprintln(v, getFeedItemTitles(next, 0))
		return nil
	})

	curFeed = next
	curTitle = 0
	return nil
}

func nextFeedUp(g *gocui.Gui, disableCurrent bool) error {
	next := curFeed - 1
	if next < 0 {
		next = len(feeds) - 1
	}

	g.Update(func(g *gocui.Gui) error {
		v, err := g.View("feedlist")
		if err != nil {
			return err
		}
		v.Clear()
		fmt.Fprintln(v, getFeedTitles(next))
		return nil
	})

	g.Update(func(g *gocui.Gui) error {
		v, err := g.View("main")
		if err != nil {
			return err
		}
		v.Clear()
		fmt.Fprintln(v, getFeedItemTitles(next, 0))
		return nil
	})

	curFeed = next
	curTitle = 0
	return nil
}

func nextTitleDown(g *gocui.Gui, disableCurrent bool) error {
	next := curTitle + 1
	if next > numberOfTitlesInSelectedFeed-1 {
		next = 0
	}

	g.Update(func(g *gocui.Gui) error {
		v, err := g.View("main")
		if err != nil {
			return err
		}
		v.Clear()
		fmt.Fprintln(v, getFeedItemTitles(curFeed, next))
		return nil
	})

	curTitle = next
	return nil
}

func nextTitleUp(g *gocui.Gui, disableCurrent bool) error {
	next := curTitle - 1
	if next < 0 {
		next = numberOfTitlesInSelectedFeed - 1
	}

	g.Update(func(g *gocui.Gui) error {
		v, err := g.View("main")
		if err != nil {
			return err
		}
		v.Clear()
		fmt.Fprintln(v, getFeedItemTitles(curFeed, next))
		return nil
	})

	curTitle = next
	return nil
}

func openBrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}
}

func openFeedLink(g *gocui.Gui, disableCurrent bool) error {
	fp := gofeed.NewParser()
	fp.UserAgent = "MyCustomAgent 1.0"
	feed, _ := fp.ParseURL(feeds[curFeed].Feed)
	openBrowser(feed.Items[curTitle].Link)
	return nil
}

func deleteFeed(g *gocui.Gui, disableCurrent bool) error {
	if len(feeds) > 0 {
		feeds = append(feeds[:curFeed], feeds[curFeed+1:]...)
		g.Update(func(g *gocui.Gui) error {
			v, err := g.View("feedlist")
			if err != nil {
				return err
			}
			v.Clear()
			if len(feeds) > 0 {
				fmt.Fprintln(v, getFeedTitles(0))
			}

			return nil
		})

		g.Update(func(g *gocui.Gui) error {
			v, err := g.View("main")
			if err != nil {
				return err
			}
			v.Clear()
			if len(feeds) > 0 {
				fmt.Fprintln(v, getFeedItemTitles(0, 0))
			}

			return nil
		})
	}
	// else {
	// 	g.Update(func(g *gocui.Gui) error {
	// 		v, err := g.View("main")
	// 		if err != nil {
	// 			return err
	// 		}
	// 		//v.Clear()
	// 		if len(feeds) > 0 {
	// 			fmt.Fprintln(v, "No Feeds.")
	// 		}

	// 		return nil
	// 	})
	// }
	curFeed = 0
	dataBytes, err := json.Marshal(feeds)
	if err != nil {
		fmt.Println(err)
	}

	err = ioutil.WriteFile("config.json", dataBytes, 0644)
	if err != nil {
		fmt.Println(err)
	}

	return nil
}

func initKeybindings(g *gocui.Gui) error {
	if err := g.SetKeybinding("", gocui.KeyCtrlX, gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("main", gocui.KeyCtrlE, gocui.ModNone, showAddFeedInput); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("addFeeds", gocui.KeyCtrlR, gocui.ModNone, handleAddFeedInput); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("", gocui.KeyCtrlD, gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			return nextFeedDown(g, true)
		}); err != nil {
		return err
	}

	if err := g.SetKeybinding("", gocui.KeyCtrlA, gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			return nextFeedUp(g, true)
		}); err != nil {
		return err
	}

	if err := g.SetKeybinding("", gocui.KeyCtrlS, gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			return nextTitleDown(g, true)
		}); err != nil {
		return err
	}

	if err := g.SetKeybinding("", gocui.KeyCtrlW, gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			return nextTitleUp(g, true)
		}); err != nil {
		return err
	}

	if err := g.SetKeybinding("", gocui.KeyCtrlSpace, gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			return openFeedLink(g, true)
		}); err != nil {
		return err
	}

	if err := g.SetKeybinding("", gocui.KeyCtrlY, gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			return deleteFeed(g, true)
		}); err != nil {
		return err
	}
	return nil
}

func initializeFeeds() {
	jsonFile, err := os.Open("config.json")
	if err != nil {
		fmt.Println(err)
	}

	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	json.Unmarshal(byteValue, &feeds)
}

func runGUI() {
	g, err := gocui.NewGui(gocui.OutputNormal, true)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()

	g.Highlight = true
	g.SelFgColor = gocui.ColorRed
	g.SelFrameColor = gocui.ColorRed

	g.SetManagerFunc(layout)
	if err := initKeybindings(g); err != nil {
		log.Panicln(err)
	}

	//g.Cursor = true
	//g.Mouse = true

	if err := g.MainLoop(); err != nil && !errors.Is(err, gocui.ErrQuit) {
		log.Panicln(err)
	}
}

func main() {
	initializeFeeds()
	runGUI()
}
