// main.go
package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "os"
    "os/exec"
    "sync"
    "time"
)

// Core PnR structures
type PnR struct {
    Prompt    string      `json:"prompt"`
    Response  []string    `json:"response"`
    TV        string      `json:"tv"` // Y, N, U (Unknown)
}

type DesignChunk struct {
    Name       string          `json:"name"`
    Gatekeeper map[string]PnR  `json:"gatekeeper"`
    Flowin     map[string]PnR  `json:"flowin"`
    Flowout    map[string]PnR  `json:"flowout"`
    Command    string          `json:"command"`
    Status     string          `json:"status"` // ready, executing, completed
}

type CPUX struct {
    ID           string        `json:"id"`
    DesignChunks []DesignChunk `json:"design_chunks"`
    RTState      map[string]PnR `json:"rt_state"` // Runtime state
}

type Domain struct {
    Name  string          `json:"name"`
    CPUXs map[string]CPUX `json:"cpuxs"`
}

// IntentionLoop manages CPUX execution
type IntentionLoop struct {
    domain *Domain
    cpux   CPUX
    mutex  sync.Mutex
}

func NewIntentionLoop(domain *Domain, cpuxID string) *IntentionLoop {
    if cpux, exists := domain.CPUXs[cpuxID]; exists {
        return &IntentionLoop{
            domain: domain,
            cpux:   cpux,
        }
    }
    return nil
}

// main.go
// ... (previous imports remain the same)

func (il *IntentionLoop) updateRTStateFromRuntime() error {
    files, err := ioutil.ReadDir("runtime")
    if err != nil {
        return fmt.Errorf("failed to read runtime directory: %v", err)
    }

    for _, file := range files {
        if file.IsDir() {
            continue
        }

        data, err := ioutil.ReadFile(fmt.Sprintf("runtime/%s", file.Name()))
        if err != nil {
            continue
        }

        var status map[string]PnR
        if err := json.Unmarshal(data, &status); err != nil {
            continue
        }

        // Update RTState with any new status
        il.mutex.Lock()
        for key, value := range status {
            il.cpux.RTState[key] = value
            fmt.Printf("Updated RTState from file %s: %s = %s\n", file.Name(), key, value.TV)
        }
        il.mutex.Unlock()
    }
    return nil
}

func (il *IntentionLoop) Execute() error {
    fmt.Println("Starting IntentionLoop execution")
    maxIterations := 30 // Add safety limit
    iteration := 0

    for {
        iteration++
        if iteration > maxIterations {
            return fmt.Errorf("exceeded maximum iterations (%d)", maxIterations)
        }

        if err := il.updateRTStateFromRuntime(); err != nil {
            return err
        }

        allCompleted := true
        anyExecuting := false

        // Check all chunks
        for i := range il.cpux.DesignChunks {
            chunk := &il.cpux.DesignChunks[i]
            
            switch chunk.Status {
            case "completed":
                continue
            case "executing":
                anyExecuting = true
                allCompleted = false
                // Check if execution is actually complete
                if il.checkChunkCompletion(chunk) {
                    chunk.Status = "completed"
                    fmt.Printf("Marked chunk as completed: %s\n", chunk.Name)
                }
            case "ready":
                allCompleted = false
                if il.checkGatekeeper(chunk) {
                    fmt.Printf("Executing chunk: %s\n", chunk.Name)
                    if err := il.executeChunk(chunk); err != nil {
                        return fmt.Errorf("chunk execution error: %v", err)
                    }
                    anyExecuting = true
                }
            }
        }

        if allCompleted {
            fmt.Println("All chunks completed successfully")
            return nil
        }

        if !anyExecuting {
            fmt.Println("No chunks executing and not all completed - stopping")
            return nil
        }

        time.Sleep(time.Second)
    }
}
//..for comp
func (il *IntentionLoop) checkChunkCompletion(chunk *DesignChunk) bool {
    il.mutex.Lock()
    defer il.mutex.Unlock()

    // Check if all flowout PnRs are present in RTState with expected values
    for prompt, expectedPnR := range chunk.Flowout {
        if rtPnR, exists := il.cpux.RTState[prompt]; !exists || rtPnR.TV != expectedPnR.TV {
            return false
        }
    }
    return true
}

// ... (rest of the code remains the same)
func (il *IntentionLoop) checkGatekeeper(chunk *DesignChunk) bool {
    il.mutex.Lock()
    defer il.mutex.Unlock()

    for prompt, pnr := range chunk.Gatekeeper {
        if rtPnr, exists := il.cpux.RTState[prompt]; !exists || rtPnr.TV != pnr.TV {
            return false
        }
    }
    return true
}

func (il *IntentionLoop) executeChunk(chunk *DesignChunk) error {
    il.mutex.Lock()
    chunk.Status = "executing"
    il.mutex.Unlock()

    fmt.Printf("Starting execution of chunk: %s\n", chunk.Name)
    cmd := exec.Command("sh", "-c", chunk.Command)
    if err := cmd.Start(); err != nil {
        chunk.Status = "ready"  // Reset status on error
        return err
    }

    // Update flowout in a goroutine
    go func() {
        cmd.Wait()
        il.mutex.Lock()
        for prompt, pnr := range chunk.Flowout {
            il.cpux.RTState[prompt] = pnr
        }
        
        // Mark as completed
        chunk.Status = "completed"
        fmt.Printf("Chunk completed: %s\n", chunk.Name)
        il.mutex.Unlock()
    }()

    return nil
}


func main() {
    // Load domain configuration
    domain := loadDomain("config/domain.json")
    
    // Start intention loop for the stack CPUX
    il := NewIntentionLoop(domain, "stack_startup")
    if il == nil {
        fmt.Println("Failed to create IntentionLoop: CPUX not found")
        os.Exit(1)
    }

    fmt.Printf("Starting CPUX: %s\n", il.cpux.ID)
    if err := il.Execute(); err != nil {
        fmt.Printf("Execution error: %v\n", err)
        os.Exit(1)
    }
    fmt.Println("CPUX execution completed")
}

func loadDomain(path string) *Domain {
    data, err := ioutil.ReadFile(path)
    if err != nil {
        panic(err)
    }

    var domain Domain
    if err := json.Unmarshal(data, &domain); err != nil {
        panic(err)
    }

    return &domain
}