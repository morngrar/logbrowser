package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"unicode"

	"github.com/gdamore/tcell/v2"
)

func drawLine(s tcell.Screen, x, y int, style tcell.Style, text string) {
	w, _ := s.Size()
	row := y
	col := x
	for _, r := range []rune(text) {
		s.SetContent(col, row, r, nil, style)
		col++
		if col >= w {
			break
		}
	}
	if col < w-1 {
		for i := col; i < w; i++ {
			s.SetContent(i, row, ' ', nil, style)
		}
	}
}

type Mark struct {
	coords Coords
	note   string
}

type Coords struct {
	x int
	y int
}

type Buffer struct {
	currentOffset Coords
	lines         []string
}

func readFileToBuf(path string) (*Buffer, error) {
	var err error
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open file for reading into buffer: %w", err)
	}
	defer file.Close()

	buf := &Buffer{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		buf.lines = append(buf.lines, scanner.Text())
	}

	return buf, nil
}

func renderViewPort(s tcell.Screen, style tcell.Style, buf *Buffer) {
	_, h := s.Size()

	if h > len(buf.lines) {
		h = len(buf.lines)
	}

	for y := 0; y < h; y++ {
		lineIndex := buf.currentOffset.y + y
		drawLine(s, buf.currentOffset.x, y, style, buf.lines[lineIndex])
	}
}

func main() {
	log.SetFlags(0) // no timestamps

	// read file from argument
	if len(os.Args) != 2 {
		fmt.Println("Usage: logbrowser FILE")
		os.Exit(1)
	}

	path := os.Args[1]
	buf, err := readFileToBuf(path)
	if err != nil {
		log.Fatalf("Couldn't read file: %s", err)
	}

	defStyle := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)

	// Initialize screen
	s, err := tcell.NewScreen()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	if err := s.Init(); err != nil {
		log.Fatalf("%+v", err)
	}
	s.SetStyle(defStyle)
	s.Clear()

	quit := func() {
		// You have to catch panics in a defer, clean up, and
		// re-raise them - otherwise your application can
		// die without leaving any diagnostic trace.
		maybePanic := recover()
		s.Fini()
		if maybePanic != nil {
			panic(maybePanic)
		}
	}
	defer quit()

	// Here's how to get the screen size when you need it.
	// xmax, ymax := s.Size()

	isExitKey := func(ev *tcell.EventKey) bool {
		switch ev.Key() {
		case tcell.KeyEscape:
			fallthrough
		case tcell.KeyCtrlC:
			return true
		}

		if unicode.ToLower(ev.Rune()) == 'q' {
			return true
		}

		return false
	}

	marks := []Mark{}
	currentMark := -1
	nextMark := func() {
		if len(marks) == 0 {
			return
		}

		currentMark++

		if currentMark >= len(marks) {
			currentMark = 0
		}
	}
	prevMark := func() {
		if len(marks) == 0 {
			return
		}

		currentMark--

		if currentMark < 0 {
			currentMark = len(marks) - 1
		}
	}

	// Event loop
	for {

		// Update screen
		renderViewPort(s, defStyle, buf)
		s.Show()

		// Poll event
		ev := s.PollEvent()

		// Process event
		switch ev := ev.(type) {
		case *tcell.EventResize:
			s.Sync()
		case *tcell.EventKey:
			if isExitKey(ev) {
				return
			} else if ev.Key() == tcell.KeyCtrlL {
				s.Sync()
			} else if ev.Rune() == ' ' {
				_, h := s.Size()
				buf.currentOffset.y += h - 1
			} else if ev.Rune() == 'b' {
				_, h := s.Size()
				buf.currentOffset.y -= h - 1
				if buf.currentOffset.y < 0 {
					buf.currentOffset.y = 0
				}
			} else if ev.Rune() == 'j' {
				buf.currentOffset.y++
				_, h := s.Size()
				if buf.currentOffset.y+h >= len(buf.lines) {
					buf.currentOffset.y = len(buf.lines) - h
				}
			} else if ev.Rune() == 'k' {
				buf.currentOffset.y--
				if buf.currentOffset.y < 0 {
					buf.currentOffset.y = 0
				}
			} else if ev.Rune() == 'J' {
				_, h := s.Size()
				buf.currentOffset.y += h / 2
				if buf.currentOffset.y+h >= len(buf.lines) {
					buf.currentOffset.y = len(buf.lines) - h
				}
			} else if ev.Key() == tcell.KeyCtrlD {
				_, h := s.Size()
				buf.currentOffset.y += h / 2
				if buf.currentOffset.y+h >= len(buf.lines) {
					buf.currentOffset.y = len(buf.lines) - h
				}
			} else if ev.Rune() == 'K' {
				_, h := s.Size()
				buf.currentOffset.y -= h / 2
				if buf.currentOffset.y < 0 {
					buf.currentOffset.y = 0
				}
			} else if ev.Key() == tcell.KeyCtrlU {
				_, h := s.Size()
				buf.currentOffset.y -= h / 2
				if buf.currentOffset.y < 0 {
					buf.currentOffset.y = 0
				}
			} else if ev.Rune() == 'h' {
				buf.currentOffset.x++
				if buf.currentOffset.x > 0 {
					buf.currentOffset.x = 0
				}
			} else if ev.Rune() == 'l' {
				buf.currentOffset.x--
			} else if ev.Rune() == 'H' {
				w, _ := s.Size()
				buf.currentOffset.x += w / 2
				if buf.currentOffset.x > 0 {
					buf.currentOffset.x = 0
				}
			} else if ev.Rune() == 'L' {
				w, _ := s.Size()
				buf.currentOffset.x -= w / 2
			} else if ev.Rune() == '0' {
				buf.currentOffset.x = 0
			} else if ev.Rune() == 'g' {
				buf.currentOffset.y = 0
			} else if ev.Rune() == 'G' {
				_, h := s.Size()
				buf.currentOffset.y = len(buf.lines) - h
			} else if ev.Rune() == 'm' {
				marks = append(marks, Mark{buf.currentOffset, ""})
				nextMark()
			} else if ev.Rune() == 'n' {
				nextMark()
				if currentMark < 0 {
					break
				}
				buf.currentOffset = marks[currentMark].coords
			} else if ev.Rune() == 'p' {
				prevMark()
				if currentMark < 0 {
					break
				}
				buf.currentOffset = marks[currentMark].coords
			}
		}
	}
}
