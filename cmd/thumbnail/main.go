package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/SeaCloudHub/backend/adapters/postgrestore"
	"github.com/SeaCloudHub/backend/adapters/redisstore"
	"github.com/SeaCloudHub/backend/adapters/services"
	"github.com/SeaCloudHub/backend/domain/file"
	"github.com/SeaCloudHub/backend/domain/identity"
	"github.com/SeaCloudHub/backend/domain/pubsub"
	"github.com/SeaCloudHub/backend/pkg/config"
	"github.com/SeaCloudHub/backend/pkg/logger"
	"github.com/SeaCloudHub/backend/pkg/sentry"
	sentrygo "github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type service struct {
	applog        *zap.SugaredLogger
	userStore     identity.Store
	fileStore     file.Store
	fileService   file.Service
	pubsubService pubsub.Service
}

type File struct {
	ID   uuid.UUID `json:"id"`
	Mime string    `json:"mime"`
}

func main() {
	applog, err := logger.NewAppLogger()
	if err != nil {
		log.Fatalf("cannot load config: %v\n", err)
	}
	// defer logger.Sync(applog)

	cfg, err := config.LoadConfig()
	if err != nil {
		applog.Fatal(err)
	}

	err = sentrygo.Init(sentrygo.ClientOptions{
		Dsn:              cfg.SentryDSN,
		Environment:      cfg.AppEnv,
		AttachStacktrace: true,
	})
	if err != nil {
		applog.Fatalf("cannot init sentry: %v", err)
	}
	defer sentrygo.Flush(sentry.FlushTime)

	db, err := postgrestore.NewConnection(postgrestore.ParseFromConfig(cfg))
	if err != nil {
		applog.Fatalf("cannot connect to db: %v\n", err)
	}

	redis, err := redisstore.NewConnection(redisstore.ParseFromConfig(cfg))
	if err != nil {
		applog.Fatal("cannot connect to redis: %v\n", err)
	}

	s := &service{
		applog:        applog,
		userStore:     postgrestore.NewUserStore(db),
		fileStore:     postgrestore.NewFileStore(db),
		fileService:   services.NewFileService(cfg),
		pubsubService: redisstore.NewRedisClient(redis),
	}

	ctx := context.Background()

	// pubsub
	pubsub := s.pubsubService.Subscribe(ctx, "thumbnails")
	defer pubsub.Close()

	// listen for messages
	for {
		msg, err := pubsub.ReceiveMessage(ctx)
		if err != nil {
			log.Fatalf("cannot receive message: %v\n", err)
		}

		// parse message
		var files []File
		if err := json.Unmarshal([]byte(msg.Payload), &files); err != nil {
			log.Fatalf("cannot unmarshal payload: %v\n", err)
		}

		// iterate over the payload
		for _, f := range files {
			log.Printf("file: %v\n", f)
			if err := s.process(ctx, &f); err != nil {
				applog.Infof("cannot process file: %v\n", err)
			}
		}

		log.Printf("message received: %s\n", msg.Payload)
	}
}

func (s *service) process(ctx context.Context, f *File) error {
	// get converter
	c := getConverter(f.Mime)
	if c == nil {
		return fmt.Errorf("converter not found for mime type: %s", f.Mime)
	}

	// download and save the file to disk
	rc, _, err := s.fileService.DownloadFile(ctx, f.ID.String())
	if err != nil {
		return fmt.Errorf("download file: %v", err)
	}
	defer rc.Close()

	mime.AddExtensionType(".mov", "video/quicktime")
	exts, _ := mime.ExtensionsByType(f.Mime)
	if len(exts) == 0 {
		return fmt.Errorf("extensions not found for mime type: %s", f.Mime)
	}

	df, err := os.Create(f.ID.String() + exts[0])
	if err != nil {
		return fmt.Errorf("create file: %v", err)
	}
	defer df.Close()

	if _, err := io.Copy(df, rc); err != nil {
		return fmt.Errorf("copy file: %v", err)
	}

	// close the file
	if err := df.Close(); err != nil {
		return fmt.Errorf("close file: %v", err)
	}

	input := f.ID.String() + exts[0]
	fileName := "thumb_" + f.ID.String() + ".png"

	// create a thumbnail
	if err := c.Convert(ctx, input, fileName); err != nil {
		return fmt.Errorf("convert: %v", err)
	}

	// upload the thumbnail to assets/thumbnails
	rc, err = os.Open(fileName)
	if err != nil {
		return fmt.Errorf("open file: %v", err)
	}
	defer rc.Close()

	fullPath := filepath.Join("/assets", "images", fileName)
	if _, err := s.fileService.CreateFile(ctx, rc, fullPath, "image/png"); err != nil {
		return fmt.Errorf("create file: %v", err)
	}

	// delete the temporary files
	if err := os.Remove(input); err != nil {
		return fmt.Errorf("remove file: %v", err)
	}

	if err := os.Remove(fileName); err != nil {
		return fmt.Errorf("remove file: %v", err)
	}

	// update the file record in the database
	if err := s.fileStore.UpdateThumbnail(ctx, f.ID, filepath.Join("/api/assets/images", fileName)); err != nil {
		return fmt.Errorf("update thumbnail: %v", err)
	}

	return nil
}

func getConverter(mime string) Converter {
	switch {
	case strings.HasPrefix(mime, "image"):
		return &ImageConverter{}
	case strings.HasPrefix(mime, "video"):
		return &VideoConverter{}
	case strings.HasPrefix(mime, "application/pdf"):
		return &PDFConverter{}
	}

	return nil
}

type Converter interface {
	Convert(ctx context.Context, input string, output string) error
}

type ImageConverter struct{}

func (c *ImageConverter) Convert(ctx context.Context, input string, output string) error {
	cmd := exec.Command("convert", input, "-resize", "200x200", "-gravity", "center", "-extent", "200x200", output)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		log.Println(cmd.String(), out.String(), stderr.String())

		return fmt.Errorf("convert: %v", err)
	}

	return nil
}

type VideoConverter struct{}

func (c *VideoConverter) Convert(ctx context.Context, input string, output string) error {
	cmd := exec.Command("ffmpeg", "-i", input, "-vf", "scale=200:200:force_original_aspect_ratio=decrease,pad=200:200:(ow-iw)/2:(oh-ih)/2", "-vframes", "1", "-update", "true", output)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		log.Println(cmd.String(), out.String(), stderr.String())

		return fmt.Errorf("convert: %v", err)
	}

	return nil
}

type PDFConverter struct{}

func (c *PDFConverter) Convert(ctx context.Context, input string, output string) error {
	cmd := exec.Command("convert", fmt.Sprintf(`%s[0]`, input), "-resize", "200x200", "-gravity", "center", "-extent", "200x200", output)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		log.Println(cmd.String(), out.String(), stderr.String())

		return fmt.Errorf("convert: %v", err)
	}

	return nil
}
