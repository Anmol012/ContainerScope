// cmd/main.go
package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var dockerClient *client.Client
var hostname string

func init() {
	var err error
	dockerClient, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	hostname = os.Getenv("HOSTNAME")
	if hostname == "" {
		hostname, _ = os.Hostname()
	}
}

// formatLogs adds line numbers to logs
func formatLogs(logs string) string {
	lines := strings.Split(logs, "\n")
	var sb strings.Builder
	for i, line := range lines {
		if line != "" {
			sb.WriteString(fmt.Sprintf("%d: %s\n\n", i+1, strings.TrimSpace(line)))
		}
	}
	return sb.String()
}

// formatImages formats the list of images
func formatImages(images []types.ImageSummary) []map[string]interface{} {
	imageList := []map[string]interface{}{}
	for _, image := range images {
		var repository, tag string
		if len(image.RepoTags) > 0 {
			parts := strings.SplitN(image.RepoTags[0], ":", 2)
			if len(parts) == 2 {
				repository = parts[0]
				tag = parts[1]
			} else {
				repository = "<none>"
				tag = "<none>"
			}
		} else {
			repository = "<none>"
			tag = "<none>"
		}

		// Skip images with <none> repository
		if repository == "<none>" || repository == "" {
			continue
		}

		createdTime := time.Unix(image.Created, 0).Format("2006-01-02 15:04:05")
		name := strings.SplitN(repository, "/", 2)
		if len(name) > 1 {
			name = name[1:]
		}

		imageInfo := map[string]interface{}{
			"node":      hostname,
			"id":        image.ID,
			"name":      name[0],
			"repository": repository,
			"tag":       tag,
			"created":   createdTime,
			"size":      fmt.Sprintf("%.2f MB", float64(image.Size)/1024/1024),
		}
		imageList = append(imageList, imageInfo)
	}
	return imageList
}

func main() {
	r := gin.Default()

	// Enable CORS
	r.Use(cors.Default())

	// List containers
	r.GET("/containers", listContainers)

	// Get container logs
	r.GET("/containers/:container_id/logs", getContainerLogs)

	// Download container logs
	r.GET("/containers/:container_id/logs/download", downloadContainerLogs)

	// Stop container
	r.POST("/containers/stop", stopContainer)

	// Start container
	r.POST("/containers/start", startContainer)

	// Restart container
	r.POST("/containers/restart", restartContainer)

	// Inspect container
	r.GET("/containers/:container_id/inspect", inspectContainer)

	// Container stats
	r.GET("/containers/:container_id/stats", containerStats)

	// Delete container
	r.DELETE("/containers/delete", deleteContainer)

	// List images
	r.GET("/images", listImages)

	r.Run(":5050")
}

func listContainers(c *gin.Context) {
	containers, err := dockerClient.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error listing containers: %v", err)})
		return
	}

	images, err := dockerClient.ImageList(context.Background(), types.ImageListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error listing images: %v", err)})
		return
	}

	imageMap := make(map[string]string)
	for _, img := range images {
		if len(img.RepoTags) > 0 {
			imageMap[img.ID] = img.RepoTags[0]
		} else {
			imageMap[img.ID] = ""
		}
	}

	containerList := []map[string]interface{}{}
	for _, cont := range containers {
		portsInfo := []string{}
		for port, bindings := range cont.Ports {
			if len(bindings) > 0 {
				externalPort := bindings[0].HostPort
				internalPort := strconv.Itoa(int(port.Int()))
				portsInfo = append(portsInfo, fmt.Sprintf("%s:%s", externalPort, internalPort))
			}
		}

		containerInfo := map[string]interface{}{
			"node":    hostname,
			"name":    cont.Names[0][1:], // Remove leading '/'
			"id":      cont.ID[:10],      // Short ID
			"running": cont.State == "running",
			"ports":   portsInfo,
			"image":   imageMap[cont.ImageID],
		}
		containerList = append(containerList, containerInfo)
	}

	c.JSON(http.StatusOK, containerList)
}

func getContainerLogs(c *gin.Context) {
	containerID := c.Param("container_id")
	linesStr := c.Query("lines")
	var lines int
	if linesStr != "" {
		var err error
		lines, err = strconv.Atoi(linesStr)
		if err != nil {
			lines = 0
		}
	}

	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       fmt.Sprintf("%d", lines),
	}

	out, err := dockerClient.ContainerLogs(context.Background(), containerID, options)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error retrieving container logs: %v", err)
		return
	}
	defer out.Close()

	logBytes, err := io.ReadAll(out)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error reading container logs: %v", err)
		return
	}

	formattedLogs := formatLogs(string(logBytes))
	c.String(http.StatusOK, formattedLogs)
}

func downloadContainerLogs(c *gin.Context) {
	containerID := c.Param("container_id")
	linesStr := c.Query("lines")
	var lines int
	if linesStr != "" {
		var err error
		lines, err = strconv.Atoi(linesStr)
		if err != nil {
			lines = 0
		}
	}

	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       fmt.Sprintf("%d", lines),
	}

	out, err := dockerClient.ContainerLogs(context.Background(), containerID, options)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error retrieving container logs: %v", err)
		return
	}
	defer out.Close()

	logBytes, err := io.ReadAll(out)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error reading container logs: %v", err)
		return
	}

	formattedLogs := formatLogs(string(logBytes))
	filename := fmt.Sprintf("container_logs_%s.txt", containerID)

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "text/plain")
	c.Data(http.StatusOK, "text/plain", []byte(formattedLogs))
}

func stopContainer(c *gin.Context) {
	var req struct {
		ContainerID string `json:"container_id"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if err := dockerClient.ContainerStop(context.Background(), req.ContainerID, container.StopOptions{}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error stopping container: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Container stopped successfully"})
}

func startContainer(c *gin.Context) {
	var req struct {
		ContainerID string `json:"container_id"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if err := dockerClient.ContainerStart(context.Background(), req.ContainerID, container.StartOptions{}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error starting container: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Container started successfully"})
}

func restartContainer(c *gin.Context) {
	var req struct {
		ContainerID string `json:"container_id"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if err := dockerClient.ContainerRestart(context.Background(), req.ContainerID, container.StopOptions{}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error restarting container: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Container restarted successfully"})
}

func inspectContainer(c *gin.Context) {
	containerID := c.Param("container_id")
	inspection, _, err := dockerClient.ContainerInspectWithRaw(context.Background(), containerID, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error inspecting container: %v", err)})
		return
	}

	c.JSON(http.StatusOK, inspection)
}

func containerStats(c *gin.Context) {
	containerID := c.Param("container_id")
	stats, err := dockerClient.ContainerStatsOneShot(context.Background(), containerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error retrieving container stats: %v", err)})
		return
	}
	defer stats.Body.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, stats.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error reading container stats: %v", err)})
		return
	}

	c.Data(http.StatusOK, "application/json", buf.Bytes())
}

func deleteContainer(c *gin.Context) {
	var req struct {
		ContainerID string `json:"container_id"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if err := dockerClient.ContainerRemove(context.Background(), req.ContainerID, container.RemoveOptions{Force: true}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error deleting container: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Container deleted successfully"})
}

func listImages(c *gin.Context) {
	images, err := dockerClient.ImageList(context.Background(), types.ImageListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error listing images: %v", err)})
		return
	}

	formattedImages := formatImages(images)
	c.JSON(http.StatusOK, formattedImages)
}
