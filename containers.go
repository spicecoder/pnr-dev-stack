// containers.go
package main

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// ContainerConfig defines Docker-specific settings for a design chunk
type ContainerConfig struct {
    Name      string            `json:"name,omitempty"`
    Image     string            `json:"image,omitempty"`
    BuildPath string            `json:"build_path,omitempty"`
    Ports     map[string]string `json:"ports,omitempty"`
    Volumes   []string          `json:"volumes,omitempty"`
    Envs      []string          `json:"env,omitempty"`
}

// ContainerDesignChunk extends DesignChunk with container configuration
type ContainerDesignChunk struct {
    DesignChunk
    Container ContainerConfig `json:"container"`
}

// ContainerIntentionLoop manages container-based CPUX execution
type ContainerIntentionLoop struct {
    domain       *Domain
    cpux        CPUX
    dockerClient *client.Client
    ctx         context.Context
    mutex       sync.Mutex
    runtimePath string
}

// NewContainerIntentionLoop creates a new container orchestrator
func NewContainerIntentionLoop(domain *Domain, cpuxID, runtimePath string) (*ContainerIntentionLoop, error) {
    cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
    if err != nil {
        return nil, fmt.Errorf("failed to create Docker client: %v", err)
    }

    // Validate CPUX exists
    cpux, exists := domain.CPUXs[cpuxID]
    if !exists {
        return nil, fmt.Errorf("CPUX %s not found in domain", cpuxID)
    }

    // Initialize RTState if nil
    if cpux.RTState == nil {
        cpux.RTState = make(map[string]PnR)
    }

    // Initialize system ready state to trigger the first container
    cpux.RTState["system_ready"] = PnR{
        Prompt:   "Is system ready?",
        Response: []string{"yes"},
        TV:      "Y",
    }

    return &ContainerIntentionLoop{
        domain:       domain,
        cpux:        cpux,
        dockerClient: cli,
        ctx:         context.Background(),
        runtimePath: runtimePath,
    }, nil
}
// checkChunkCompletion verifies if a chunk has completed its execution
func (il *ContainerIntentionLoop) checkChunkCompletion(chunk *DesignChunk) bool {
    il.mutex.Lock()
    defer il.mutex.Unlock()

    fmt.Printf("Checking completion for %s\n", chunk.Name)
    for prompt, expectedPnR := range chunk.Flowout {
        if rtPnR, exists := il.cpux.RTState[prompt]; !exists {
            fmt.Printf("  Missing expected output: %s\n", prompt)
            return false
        } else if rtPnR.TV != expectedPnR.TV {
            fmt.Printf("  Unexpected value for %s: got %s, want %s\n", 
                prompt, rtPnR.TV, expectedPnR.TV)
            return false
        }
    }
    return true
}

func (il *ContainerIntentionLoop) checkGatekeeper(chunk *DesignChunk) bool {
    il.mutex.Lock()
    defer il.mutex.Unlock()

    fmt.Printf("Checking gatekeeper for %s\n", chunk.Name)
    for prompt, pnr := range chunk.Gatekeeper {
        if rtPnr, exists := il.cpux.RTState[prompt]; !exists {
            fmt.Printf("  Missing gatekeeper condition: %s\n", prompt)
            return false
        } else if rtPnr.TV != pnr.TV {
            fmt.Printf("  Gatekeeper condition not met for %s: got %s, want %s\n", 
                prompt, rtPnr.TV, pnr.TV)
            return false
        }
    }
    return true
}

// // checkGatekeeper verifies if a chunk's prerequisites are met
// func (il *ContainerIntentionLoop) checkGatekeeper(chunk *DesignChunk) bool {
//     il.mutex.Lock()
//     defer il.mutex.Unlock()

//     for prompt, pnr := range chunk.Gatekeeper {
//         if rtPnr, exists := il.cpux.RTState[prompt]; !exists || rtPnr.TV != pnr.TV {
//             return false
//         }
//     }
//     return true
// }

// updateRTStateFromRuntime reads and updates runtime state from files
func (il *ContainerIntentionLoop) updateRTStateFromRuntime() error {
    files, err := ioutil.ReadDir(il.runtimePath)
    if err != nil {
        return fmt.Errorf("failed to read runtime directory: %v", err)
    }

    for _, file := range files {
        if file.IsDir() {
            continue
        }

        data, err := ioutil.ReadFile(filepath.Join(il.runtimePath, file.Name()))
        if err != nil {
            continue
        }

        var status map[string]PnR
        if err := json.Unmarshal(data, &status); err != nil {
            continue
        }

        il.mutex.Lock()
        for key, value := range status {
            il.cpux.RTState[key] = value
        }
        il.mutex.Unlock()
    }
    return nil
}

// getTarContext creates a tar archive for Docker build context
func getTarContext(contextPath string) io.Reader {
    ctx := new(bytes.Buffer)
    tw := tar.NewWriter(ctx)
    defer tw.Close()

    if err := filepath.Walk(contextPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        var header *tar.Header
        if info.IsDir() {
            header = &tar.Header{
                Name:     path,
                Mode:     0755,
                Typeflag: tar.TypeDir,
                ModTime:  info.ModTime(),
                Size:     info.Size(),
            }
        } else {
            header = &tar.Header{
                Name:     path,
                Mode:     0644,
                Typeflag: tar.TypeReg,
                ModTime:  info.ModTime(),
                Size:     info.Size(),
            }
        }

        relPath, err := filepath.Rel(contextPath, path)
        if err != nil {
            return err
        }
        header.Name = relPath

        if err := tw.WriteHeader(header); err != nil {
            return err
        }

        if !info.IsDir() {
            file, err := os.Open(path)
            if err != nil {
                return err
            }
            defer file.Close()

            _, err = io.Copy(tw, file)
            return err
        }
        return nil
    }); err != nil {
        return nil
    }

    return ctx
}

// Execute runs the container orchestration
func (il *ContainerIntentionLoop) Execute() error {
    fmt.Println("Starting PnR Container Orchestration")

    if err := il.cleanup(); err != nil {
        return fmt.Errorf("cleanup failed: %v", err)
    }

    _, err := il.dockerClient.NetworkCreate(il.ctx, "pnr_network", types.NetworkCreate{})
    if err != nil {
        return fmt.Errorf("failed to create network: %v", err)
    }

    for {
        if err := il.updateRTStateFromRuntime(); err != nil {
            return err
        }
        
        fmt.Println("\nCurrent RTState:")
        for k, v := range il.cpux.RTState {
            fmt.Printf("%s: %s\n", k, v.TV)
        }

        allCompleted := true
        anyExecuting := false

        for i := range il.cpux.DesignChunks {
            chunk := &il.cpux.DesignChunks[i]
            
            fmt.Printf("\nProcessing chunk: %s (status: %s)\n", chunk.Name, chunk.Status)

            switch chunk.Status {
            case "completed":
                fmt.Printf("Chunk %s is completed\n", chunk.Name)
                continue
                
            case "executing":
                anyExecuting = true
                allCompleted = false
                if il.checkChunkCompletion(chunk) {
                    chunk.Status = "completed"
                    fmt.Printf("Chunk %s completed successfully\n", chunk.Name)
                }
                
            case "ready":
                allCompleted = false
                gatekeeperMet := il.checkGatekeeper(chunk)
                fmt.Printf("Gatekeeper check for %s: %v\n", chunk.Name, gatekeeperMet)
                
                if gatekeeperMet {
                    fmt.Printf("Container config for %s:\n", chunk.Name)
                    fmt.Printf("  Name: %s\n", chunk.Container.Name)
                    fmt.Printf("  Image: %s\n", chunk.Container.Image)
                    fmt.Printf("  Ports: %v\n", chunk.Container.Ports)
                    
                    if err := il.startContainer(chunk); err != nil {
                        return fmt.Errorf("failed to start container %s: %v", 
                            chunk.Container.Name, err)
                    }
                    chunk.Status = "executing"
                    anyExecuting = true
                }
        }

        if allCompleted {
            fmt.Println("All containers completed successfully")
            return nil
        }

        if !anyExecuting {
            fmt.Println("No containers executing and not all completed - stopping")
            return nil
        }

        time.Sleep(time.Second)
    }
}
}
func (il *ContainerIntentionLoop) cleanup() error {
    // Remove existing containers
    containers, err := il.dockerClient.ContainerList(il.ctx, container.ListOptions{All: true})
    if err != nil {
        return fmt.Errorf("failed to list containers: %v", err)
    }

    for _, cont := range containers {
        if err := il.dockerClient.ContainerRemove(il.ctx, cont.ID, container.RemoveOptions{Force: true}); err != nil {
            fmt.Printf("Warning: failed to remove container %s: %v\n", cont.ID, err)
        }
    }

    // Remove network
    networks, err := il.dockerClient.NetworkList(il.ctx, network.ListOptions{})
    if err != nil {
        return fmt.Errorf("failed to list networks: %v", err)
    }

    for _, network := range networks {
        if network.Name == "pnr_network" {
            if err := il.dockerClient.NetworkRemove(il.ctx, network.ID); err != nil {
                fmt.Printf("Warning: failed to remove network: %v\n", err)
            }
            break
        }
    }

    return nil
}
// startContainer creates and starts a Docker container
// func getDockerClient() (*client.Client, error) {
//     return client.NewClientWithOpts(client.FromEnv)
// }

func pullImage(ctx context.Context, cli *client.Client, imageName string) error {
    reader, err := cli.ImagePull(
        ctx, 
        imageName, 
        image.PullOptions{
            All:          false,
            RegistryAuth: "",
            PrivilegeFunc: func(ctx context.Context) (string, error) {
                return "", nil  // Return empty auth string and no error
            },
            Platform: "",
        },  
    )
    if err != nil {
        return fmt.Errorf("failed to pull image %s: %v", imageName, err)
    }
    defer reader.Close()

    // Must read the reader to wait for pull to complete
    _, err = io.Copy(os.Stdout, reader)
    return err
}
// startContainer creates and starts a Docker container
func (il *ContainerIntentionLoop) startContainer(chunk *DesignChunk) error {
    fmt.Printf("Attempting to start container:\n")
    fmt.Printf("  Name: %s\n", chunk.Container.Name)
    fmt.Printf("  Image: %s\n", chunk.Container.Image)
    fmt.Printf("  BuildPath: %s\n", chunk.Container.BuildPath)

    // // Pull image if it doesn't exist
    // // Only pull for non-build images
    // if chunk.Container.BuildPath == "" {
    //     fmt.Printf("Pulling image: %s\n", chunk.Container.Image)
    //     if err := pullImage(il.ctx, il.dockerClient, chunk.Container.Image); err != nil {
    //         return err
    //     }
    // }
     // Build or pull image
     if chunk.Container.BuildPath != "" {
        fmt.Printf("Building image from %s\n", chunk.Container.BuildPath)
        buildContext := filepath.Join(".", chunk.Container.BuildPath)
        
        // Create tar of build context
        buildCtx := getTarContext(buildContext)
        if buildCtx == nil {
            return fmt.Errorf("failed to create build context from %s", buildContext)
        }
        
        // Build the image
        resp, err := il.dockerClient.ImageBuild(
            il.ctx,
            buildCtx,
            types.ImageBuildOptions{
                Tags:       []string{chunk.Container.Image},
                Dockerfile: "Dockerfile",
            },
        )
        if err != nil {
            return fmt.Errorf("build failed: %v", err)
        }
        defer resp.Body.Close()
        
        // Print build output
        io.Copy(os.Stdout, resp.Body)
    } else {
        // if chunk.Container.BuildPath == "" {
        fmt.Printf("Pulling image: %s\n", chunk.Container.Image)
        if err := pullImage(il.ctx, il.dockerClient, chunk.Container.Image); err != nil {
            return err
        }
    }
        // Pull image for non-build containers
        // reader, err := il.dockerClient.ImagePull(il.ctx, chunk.Container.Image, types.ImagePullOptions{})
        // if err != nil {
        //     return fmt.Errorf("failed to pull image: %v", err)
        // }
        // io.Copy(os.Stdout, reader)
        // reader.Close()
    

    // Get absolute path for runtime directory
    absRuntimePath, err := filepath.Abs(il.runtimePath)
    if err != nil {
        return fmt.Errorf("failed to get absolute runtime path: %v", err)
    }

    // Container configuration
    config := &container.Config{
        Image: chunk.Container.Image,
        Env:   chunk.Container.Envs,
    }

    // Port bindings
    portBindings := nat.PortMap{}
    if len(chunk.Container.Ports) > 0 {
        for containerPort, hostPort := range chunk.Container.Ports {
            port := nat.Port(fmt.Sprintf("%s/tcp", containerPort))
            portBindings[port] = []nat.PortBinding{
                {
                    HostIP:   "0.0.0.0",
                    HostPort: hostPort,
                },
            }
        }
    }
   // var absRuntimePath string;
    absRuntimePath , err1 := filepath.Abs(il.runtimePath)
    if err1 != nil {
        return fmt.Errorf("failed to get absolute runtime path: %v", err)
    }

    absConfigPath, err := filepath.Abs("config")
    if err != nil {
        return fmt.Errorf("failed to get absolute config path: %v", err)
    }

    hostConfig := &container.HostConfig{
        Mounts: []mount.Mount{
            {
                Type:   mount.TypeBind,
                Source: absRuntimePath,
                Target: "/runtime",
            },
            {
                Type:   mount.TypeBind,
                Source: absConfigPath,
                Target: "/config",
            },
        },
        PortBindings: portBindings,
    }
    fmt.Printf("Creating container %s\n", chunk.Container.Name)

    resp, err := il.dockerClient.ContainerCreate(
        il.ctx,
        config,
        hostConfig,
        nil,
        nil,
        chunk.Container.Name,
    )
    if err != nil {
        return fmt.Errorf("container creation failed: %v", err)
    }

    if err := il.dockerClient.NetworkConnect(
        il.ctx,
        "pnr_network",
        resp.ID,
        nil,
    ); err != nil {
        return fmt.Errorf("network connection failed: %v", err)
    }
    if chunk.Name == "mongodb" {
        // For MongoDB, set initial status after container starts
        status := map[string]PnR{
            "mongodb_status": {
                Prompt:   "Is MongoDB running?",
                Response: []string{"yes"},
                TV:      "Y",
            },
        }
        
        // Write to runtime directory
        statusFile := filepath.Join(il.runtimePath, "mongodb_status.json")
        statusData, _ := json.MarshalIndent(status, "", "  ")
        if err := ioutil.WriteFile(statusFile, statusData, 0644); err != nil {
            fmt.Printf("Warning: failed to write MongoDB status: %v\n", err)
        }
    }
   // Start the container
   if err := il.dockerClient.ContainerStart(il.ctx, resp.ID, container.StartOptions{}); err != nil {
    return fmt.Errorf("container start failed: %v", err)
}
    return il.dockerClient.ContainerStart(il.ctx, resp.ID, container.StartOptions{})
}