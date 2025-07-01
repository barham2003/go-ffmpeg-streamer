package main

import (
	"io"
	"log"
	"net/http"
	"os/exec"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	// Stream video endpoint
	router.GET("/stream", streamHandler)

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	log.Println("Starting server on :8080")
	router.Run(":8080")
}

func streamHandler(c *gin.Context) {

	path := c.DefaultQuery("path", "test2.mkv")
	c.Header("Content-Type", "video/mp4")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// Create ffmpeg command
	cmd := exec.Command("ffmpeg",
		"-i", path,
		"-c:v", "copy",
		"-c:a", "copy",
		"-f", "mp4",
		"-movflags", "frag_keyframe+empty_moov+default_base_moof",
		"-fflags", "+genpts",
		"pipe:1")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create stdout pipe"})
		return
	}

	if err := cmd.Start(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start ffmpeg"})
		return
	}

	// Ensure cleanup
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		cmd.Wait()
	}()

	// Stream the video data
	c.Stream(func(w io.Writer) bool {
		buffer := make([]byte, 32768) // Larger buffer for better performance
		n, err := stdout.Read(buffer)
		if err != nil {
			if err == io.EOF {
				log.Println("Stream ended")
				return false
			}
			log.Printf("Error reading from ffmpeg: %v", err)
			return false
		}

		if n > 0 {
			_, writeErr := w.Write(buffer[:n])
			if writeErr != nil {
				log.Printf("Error writing to client: %v", writeErr)
				return false
			}
		}

		return true
	})
}

// Alternative implementation using io.Copy for simpler streaming
func streamHandlerSimple(c *gin.Context) {
	c.Header("Content-Type", "video/mp4")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	cmd := exec.Command("ffmpeg",
		"-i", "test2.mkv",
		"-c:v", "copy",
		"-c:a", "copy",
		"-f", "mp4",
		"-movflags", "frag_keyframe+empty_moov",
		"pipe:1")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create pipe"})
		return
	}

	if err := cmd.Start(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start ffmpeg"})
		return
	}

	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		cmd.Wait()
	}()

	// Copy directly from ffmpeg stdout to HTTP response
	_, err = io.Copy(c.Writer, stdout)
	if err != nil {
		log.Printf("Error copying stream: %v", err)
	}
}
