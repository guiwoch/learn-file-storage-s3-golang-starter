package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	const gigabyte int64 = 1 << 30
	r.Body = http.MaxBytesReader(w, r.Body, gigabyte)

	videoIDString := r.PathValue("videoID")
	videoUUID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid videoID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Could not authenticate user", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Could not validate JWT", err)
		return
	}

	video, err := cfg.db.GetVideo(videoUUID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not get video from database", err)
		return
	}

	if video.CreateVideoParams.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "The selected user is not the owner of the video", err)
		return
	}

	file, fileHeader, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Formfile problem", err)
		return
	}
	defer file.Close()

	fileContentType := fileHeader.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(fileContentType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not get mediaType.", err)
		return
	}

	if mediaType != "video/mp4" {
		respondWithJSON(w, http.StatusBadRequest, "Invalid video format.")
		return
	}

	tempFile, err := os.CreateTemp("", "tubely-video-temp")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not create temp file.", err)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, file); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not copy file to temp location.", err)
		return
	}
	tempFile.Seek(0, io.SeekStart)

	random := make([]byte, 32)
	rand.Read(random)

	uploadFileName := base64.RawURLEncoding.EncodeToString(random)
	cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &uploadFileName,
		Body:        tempFile,
		ContentType: &mediaType,
	})
	videoURL := fmt.Sprintf(
		"http://%v.s3.%v.amazonaws.com/%v", cfg.s3Bucket, cfg.s3Region, uploadFileName)

	video.VideoURL = &videoURL

	cfg.db.UpdateVideo(video)
}
