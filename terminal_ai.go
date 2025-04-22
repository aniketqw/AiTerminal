package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/joho/godotenv"
)

// Load environment variables from .env
func init() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file:", err)
	}

	if os.Getenv("GROQ_API_KEY") == "" {
		log.Fatal("GROQ_API_KEY environment variable is not set")
	}
}

type AIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// Process AI request by sending the query to the Groq API
func processAIRequest(prompt string, systemContext string) (string, error) {
	apiURL := "https://api.groq.com/openai/v1/chat/completions"
	
	messages := []map[string]string{
		{"role": "system", "content": systemContext},
		{"role": "user", "content": prompt},
	}
	
	reqBody, _ := json.Marshal(map[string]interface{}{
		"model": "llama3-8b-8192",
		"messages": messages,
	})
	
	// Create a new request
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
	
	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+os.Getenv("GROQ_API_KEY"))
	
	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var aiResponse AIResponse
	if err := json.NewDecoder(resp.Body).Decode(&aiResponse); err != nil {
		return "", err
	}
	
	if len(aiResponse.Choices) == 0 {
		return "", fmt.Errorf("no response from AI")
	}
	
	return aiResponse.Choices[0].Message.Content, nil
}

// Execute a command and return its output
func executeCommand(command string) (string, error) {
	var cmd *exec.Cmd
	
	if runtime.GOOS == "windows" {
		cmd = exec.Command("powershell", "-Command", command)
	} else {
		cmd = exec.Command("bash", "-c", command)
	}
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), err
	}
	
	return string(output), nil
}

// Get filesystem information for the current directory
func getFilesystemInfo() (string, error) {
	var info strings.Builder
	
	// Get current directory
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	info.WriteString(fmt.Sprintf("Current directory: %s\n\n", currentDir))
	
	// List files in current directory
	files, err := ioutil.ReadDir(".")
	if err != nil {
		return "", err
	}
	
	info.WriteString("Files and directories:\n")
	for _, file := range files {
		fileType := "File"
		if file.IsDir() {
			fileType = "Directory"
		}
		info.WriteString(fmt.Sprintf("- %s (%s, %d bytes)\n", file.Name(), fileType, file.Size()))
	}
	
	// Check if this is a git repository
	if _, err := os.Stat(".git"); err == nil {
		info.WriteString("\nThis is a git repository.\n")
		
		// Get git status
		gitStatus, err := executeCommand("git status")
		if err == nil {
			info.WriteString("Git status summary:\n")
			info.WriteString(gitStatus)
		}
	}
	
	// Check if this is a Go project
	goFiles, err := filepath.Glob("*.go")
	if err == nil && len(goFiles) > 0 {
		info.WriteString("\nThis is a Go project.\n")
		info.WriteString("Go files found: " + strings.Join(goFiles, ", ") + "\n")
		
		// Check for go.mod
		if _, err := os.Stat("go.mod"); err == nil {
			goMod, err := ioutil.ReadFile("go.mod")
			if err == nil {
				info.WriteString("\ngo.mod content:\n")
				info.WriteString(string(goMod))
			}
		}
	}
	
	return info.String(), nil
}

func initTerminalUI() tcell.Screen {
	screen, err := tcell.NewScreen()
	if err != nil {
		log.Fatal(err)
	}
	if err := screen.Init(); err != nil {
		log.Fatal(err)
	}
	return screen
}

// Display text at a specific row in the terminal
func displayText(screen tcell.Screen, row int, text string) {
	// Clear the row first
	width, _ := screen.Size()
	for i := 0; i < width; i++ {
		screen.SetContent(i, row, ' ', nil, tcell.StyleDefault)
	}
	
	// Draw the text
	for i, r := range text {
		screen.SetContent(i, row, r, nil, tcell.StyleDefault)
	}
}

// Display multi-line text
func displayMultilineText(screen tcell.Screen, startRow int, text string, maxWidth int) int {
	lines := strings.Split(text, "\n")
	currentRow := startRow
	
	for _, line := range lines {
		// Word wrap long lines
		if len(line) > maxWidth {
			words := strings.Fields(line)
			currentLine := ""
			
			for _, word := range words {
				if len(currentLine)+len(word)+1 > maxWidth {
					displayText(screen, currentRow, currentLine)
					currentRow++
					currentLine = word
				} else {
					if currentLine == "" {
						currentLine = word
					} else {
						currentLine += " " + word
					}
				}
			}
			
			if currentLine != "" {
				displayText(screen, currentRow, currentLine)
				currentRow++
			}
		} else {
			displayText(screen, currentRow, line)
			currentRow++
		}
	}
	
	return currentRow
}

func main() {
	screen := initTerminalUI()
	defer screen.Fini()

	screen.Clear()
	displayText(screen, 0, "Terminal AI Assistant (Type 'exit' to quit)")
	displayText(screen, 1, "Commands: 'chat <question>', 'terminal <question>', 'run <command>'")
	displayText(screen, 2, "> ")
	cursorPos := 2 // Starting cursor position
	currentRow := 2
	inputText := ""
	outputRow := 4 // Row to display AI responses
	
	commandHistory := []string{} // Store previous command outputs
	
	screen.Show()

	for {
		ev := screen.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyCtrlC {
				return
			} else if ev.Key() == tcell.KeyEnter {
				// Process the entered text
				trimmedInput := strings.TrimSpace(inputText)
				if trimmedInput == "exit" {
					return
				} else if strings.HasPrefix(trimmedInput, "chat ") {
					query := strings.TrimPrefix(trimmedInput, "chat ")
					if query == "" {
						displayText(screen, outputRow, "Error: Please provide a question after 'chat'")
						outputRow++
					} else {
						// Show that we're processing
						displayText(screen, outputRow, "Processing your request...")
						screen.Show()
						
						// Send to AI
						response, err := processAIRequest(query, "You are a helpful assistant.")
						screen.Clear()
						displayText(screen, 0, "Terminal AI Assistant (Type 'exit' to quit)")
						displayText(screen, 1, "Commands: 'chat <question>', 'terminal <question>', 'run <command>'")
						displayText(screen, currentRow, "> "+inputText)
						
						if err != nil {
							displayText(screen, outputRow, fmt.Sprintf("Error: %v", err))
							outputRow++
						} else {
							// Display the response
							displayText(screen, outputRow, "AI Response:")
							outputRow++
							
							// Display multi-line response
							maxWidth, _ := screen.Size()
							outputRow = displayMultilineText(screen, outputRow, response, maxWidth-1)
						}
					}
				} else if strings.HasPrefix(trimmedInput, "terminal ") {
					query := strings.TrimPrefix(trimmedInput, "terminal ")
					if query == "" {
						displayText(screen, outputRow, "Error: Please provide a question after 'terminal'")
						outputRow++
					} else {
						// Show that we're processing
						displayText(screen, outputRow, "Analyzing filesystem and processing your request...")
						screen.Show()
						
						// Get filesystem info
						fsInfo, err := getFilesystemInfo()
						if err != nil {
							displayText(screen, outputRow, fmt.Sprintf("Error getting filesystem info: %v", err))
							outputRow++
						} else {
							// Create context with filesystem info and command history
							systemContext := "You are a terminal assistant that helps users navigate their filesystem and suggests commands. " +
								"When suggesting commands, be specific and explain what each command does. " +
								"Here is information about the current filesystem:\n\n" + fsInfo
							
							// Add command history context if available
							if len(commandHistory) > 0 {
								systemContext += "\n\nRecent command output history:\n" + strings.Join(commandHistory, "\n---\n")
							}
							
							// Send to AI
							response, err := processAIRequest(query, systemContext)
							screen.Clear()
							displayText(screen, 0, "Terminal AI Assistant (Type 'exit' to quit)")
							displayText(screen, 1, "Commands: 'chat <question>', 'terminal <question>', 'run <command>'")
							displayText(screen, currentRow, "> "+inputText)
							if err != nil {
								displayText(screen, outputRow, fmt.Sprintf("Error: %v", err))
								outputRow++
							} else {
								// Display the response
								displayText(screen, outputRow, "Terminal Assistant Response:")
								outputRow++
								
								// Display multi-line response
								maxWidth, _ := screen.Size()
								outputRow = displayMultilineText(screen, outputRow, response, maxWidth-1)
							}
						}
					}
				} else if strings.HasPrefix(trimmedInput, "run ") {
					command := strings.TrimPrefix(trimmedInput, "run ")
					if command == "" {
						displayText(screen, outputRow, "Error: Please provide a command after 'run'")
						outputRow++
					} else {
						// Show that we're processing
						displayText(screen, outputRow, "Executing command: " + command)
						outputRow++
						screen.Show()
						
						// Execute the command
						output, err := executeCommand(command)
						if err != nil {
							displayText(screen, outputRow, fmt.Sprintf("Command error: %v", err))
							outputRow++
						}
						
						// Store command output in history (limit to last 3 commands)
						if len(commandHistory) >= 3 {
							commandHistory = commandHistory[1:]
						}
						commandHistory = append(commandHistory, "Command: "+command+"\nOutput: "+output)
						
						// Display the output
						displayText(screen, outputRow, "Command output:")
						outputRow++
						
						// Display multi-line output
						maxWidth, _ := screen.Size()
						outputRow = displayMultilineText(screen, outputRow, output, maxWidth-1)
					}
				} else {
					displayText(screen, outputRow, "Unknown command. Use 'chat <question>', 'terminal <question>', or 'run <command>'")
					outputRow++
				}
				
				// Reset for new input
				inputText = ""
				currentRow = outputRow + 1
				
				// Check if we need to clear screen due to approaching bottom
				maxHeight, _ := screen.Size()
				if currentRow > maxHeight-5 {
					screen.Clear()
					displayText(screen, 0, "Terminal AI Assistant (Type 'exit' to quit)")
					displayText(screen, 1, "Commands: 'chat <question>', 'terminal <question>', 'run <command>'")
					currentRow = 2
					outputRow = 4
				}
				
				displayText(screen, currentRow, "> ")
				cursorPos = 2
				outputRow = currentRow + 1
				
			} else if ev.Key() == tcell.KeyBackspace || ev.Key() == tcell.KeyBackspace2 {
				if len(inputText) > 0 {
					inputText = inputText[:len(inputText)-1]
					cursorPos--
					// Clear the character
					screen.SetContent(cursorPos, currentRow, ' ', nil, tcell.StyleDefault)
				}
			} else if ev.Rune() != 0 {
				// Add the character to the input
				inputText += string(ev.Rune())
				screen.SetContent(cursorPos, currentRow, ev.Rune(), nil, tcell.StyleDefault)
				cursorPos++
			}
		}
		
		screen.Show()
	}
}
