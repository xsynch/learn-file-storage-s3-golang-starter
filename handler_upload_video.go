package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"


	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {

	const maxMemory = 1 << 30
	r.ParseMultipartForm(maxMemory)
	r.Body = http.MaxBytesReader(w, r.Body, maxMemory)

	videoIDString := r.PathValue("videoID")
	vidID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error parsing the videoid", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	video, err := cfg.db.GetVideo(vidID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error getting video information", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Invalid User", nil)
		return
	}

	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error Parsing Media Type", err)
		return
	}
	if mediaType == "" {
		respondWithError(w, http.StatusBadRequest, "Content type is empty", nil)
		return
	}
	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Only Mp4s Allowed", nil)
		return
	}

	tempFile, err := os.CreateTemp("", "tubely-temp.mp4")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error creating temporary file", err)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()
	_, err = io.Copy(tempFile, file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error Copying file to temporary file", err)
		return
	}
	_, err = tempFile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error Resetting to the start of the file", err)
		return
	}
	processedFilePath, err := processVideoForFastStart(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}
	processedFile, err := os.Open(processedFilePath)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error opening processed file", err)
		return
	}
	defer processedFile.Close()
	defer os.Remove(processedFile.Name())

	// aspectRatio,err := getVideoAspectRatio(tempFile.Name())
	aspectRatio, err := getVideoAspectRatio(processedFile.Name())
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error getting Aspect Ratio", err)
		return
	}

	c := 32
	b := make([]byte, c)
	_, err = rand.Read(b)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error generating random number", err)
		return
	}
	fileName := base64.RawURLEncoding.EncodeToString(b)
	var prefix string
	if aspectRatio == "16:9" {
		prefix = "landscape"
	} else if aspectRatio == "9:16" {
		prefix = "portrait"
	} else {
		prefix = "other"
	}

	fullName := fmt.Sprintf("%s/%s.mp4", prefix, fileName)
	// video, err = cfg.dbVideoToSignedVideo(video)
	// if err != nil {
	// 	respondWithError(w, http.StatusBadRequest, "error getting presigned video", err)
	// 	return
	// }

	_, err = cfg.s3client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket:      aws.String(cfg.s3Bucket),
		Key:         aws.String(fullName),
		Body:        processedFile,
		ContentType: aws.String(mediaType),
	})
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error uploading object", err)
		return
	}
	newVidURL := fmt.Sprintf("%s/%s",cfg.s3CfDistribution,fullName)
	// newVidURL := fmt.Sprintf("%s,%s", cfg.s3Bucket, fullName)

	video.VideoURL = &newVidURL
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error updating the video", err)
		return
	}

	// video, err = cfg.dbVideoToSignedVideo(video)
	// if err != nil {
	// 	respondWithError(w, http.StatusInternalServerError, "Error getting signed video", err)
	// 	return
	// }

	respondWithJSON(w, http.StatusOK, video)

}

// func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
// 	if video.VideoURL == nil {
// 		return video, nil
// 	}
// 	val := strings.Split(*video.VideoURL, ",")
// 	url, err := generatePresignedURL(cfg.s3client, val[0], val[1], 5*time.Minute)
// 	if err != nil {
// 		return video, err
// 	}
// 	video.VideoURL = &url

// 	return video, nil
// }
