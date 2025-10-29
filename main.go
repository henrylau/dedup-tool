package main

import (
	"flag"
	"fmt"
	"folder-similarity/core"
	"folder-similarity/ui"
	logui "folder-similarity/ui/log"
	"log"

	tea "github.com/charmbracelet/bubbletea"
)

var rootPath string

func main() {
	flag.StringVar(&rootPath, "path", "", "root path")
	flag.Parse()

	if rootPath == "" {
		rootPath = flag.Arg(0)
		if rootPath == "" {
			log.Fatal("root path is required")
		}
	}

	storage := core.NewMemoryStorage()
	logChan := make(chan string)

	scanner := core.Scanner{
		Storage: storage,
		Path:    []string{rootPath},
		Logger: func(message string) {
			logChan <- message
		},
	}

	go func() {
		count := 0
		for message := range logChan {
			count++
			if count%10 == 0 {
				fmt.Printf("[%d] %s\n", count, message)
			}
		}
	}()

	err := scanner.Scan()
	if err != nil {
		log.Fatal(err)
	}
	close(logChan)

	// Initialize the main model
	m := ui.NewMainModel()
	p := tea.NewProgram(m, tea.WithAltScreen())

	// Set up logger
	m.SetLogger(logui.EventLogger(p))

	// Initialize storage and scan folder
	m.SetStorage(storage)
	m.SetRootPath(rootPath)
	// err := core.ScanFolder(context.Background(), m.GetRootPath(), m.GetStorage())
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// Initialize similarity checker
	similarityChecker := &core.SimilarityChecker{}
	similarityChecker.CalculateSimilarity(m.GetStorage())
	m.SetSimilarityChecker(similarityChecker)

	// Set up root folder
	root, err := m.GetStorage().GetFolder(".")
	if err != nil {
		log.Fatal(err)
	}
	rootFolder := &ui.FolderItemWrapper{Folder: root}
	m.SetRootFolder(rootFolder)

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
