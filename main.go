// main.go
package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "os"
    "path/filepath"
    
)

// Core PnR structures
type PnR struct {
    Prompt    string   `json:"prompt"`
    Response  []string `json:"response"`
    TV        string   `json:"tv"`
}
// main.go
type DesignChunk struct {
    Name       string          `json:"name"`
    Gatekeeper map[string]PnR  `json:"gatekeeper"`
    Flowin     map[string]PnR  `json:"flowin"`
    Flowout    map[string]PnR  `json:"flowout"`
    Status     string          `json:"status"`
    Container  ContainerConfig `json:"container"` // Add this field
}

type CPUX struct {
    ID           string        `json:"id"`
    DesignChunks []DesignChunk `json:"design_chunks"`
    RTState      map[string]PnR `json:"rt_state"`
}

type Domain struct {
    Name  string          `json:"name"`
    CPUXs map[string]CPUX `json:"cpuxs"`
}

func loadDomain(path string) *Domain {
    data, err := ioutil.ReadFile(path)
    if err != nil {
        panic(fmt.Sprintf("Failed to read domain config: %v", err))
    }

    var domain Domain
    if err := json.Unmarshal(data, &domain); err != nil {
        panic(fmt.Sprintf("Failed to parse domain config: %v", err))
    }
    // Initialize RTState for each CPUX
    for cpuxID := range domain.CPUXs {
        cpux := domain.CPUXs[cpuxID]
        if cpux.RTState == nil {
            cpux.RTState = make(map[string]PnR)
        }
        domain.CPUXs[cpuxID] = cpux
    }

    return &domain
}

func main() {
    // Create runtime directory
    runtimePath := filepath.Join(".", "runtime")
    if err := os.MkdirAll(runtimePath, 0755); err != nil {
        fmt.Printf("Failed to create runtime directory: %v\n", err)
        os.Exit(1)
    }

    // Load domain configuration
    domain := loadDomain("config/domain.json")
    
    // Initialize container intention loop
    il, err := NewContainerIntentionLoop(domain, "stack_startup", runtimePath)
    if err != nil {
        fmt.Printf("Failed to create ContainerIntentionLoop: %v\n", err)
        os.Exit(1)
    }

    // Execute CPUX
    if err := il.Execute(); err != nil {
        fmt.Printf("Execution error: %v\n", err)
        os.Exit(1)
    }

    fmt.Println("CPUX execution completed successfully")
}