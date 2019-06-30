package main

import (
	"bytes"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/radovskyb/watcher"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	syncEnvironments()

	w := watcher.New()
	w.FilterOps(watcher.Rename, watcher.Remove, watcher.Create, watcher.Write)

	go func() {
		for {
			select {
			case event := <-w.Event:

				otherEvent := false
				if event.Op == watcher.Remove {
					otherEvent = true
					deleteFileFromS3(event.FileInfo.Name())
				}

				if event.Op == watcher.Create {
					otherEvent = true
					uploadFile(event.FileInfo.Name())
				}

				if event.Op == watcher.Rename {
					otherEvent = true
					deleteFileFromS3(event.FileInfo.Name())
					paths := strings.Split(event.Path, "->")
					splitPath := strings.Split(paths[1], "/")
					newFileName := splitPath[len(splitPath) -1]
					uploadFile(newFileName)
				}

				if event.Op == watcher.Write && !otherEvent && !event.IsDir(){
					uploadFile(event.FileInfo.Name())
				}

				log.Println(event) // Print the event's info.
			case err := <-w.Error:
				log.Fatalln(err)
			case <-w.Closed:
				return
			}
		}
	}()

	if err := w.Add("test"); err != nil {
		log.Fatalln(err)
	}

	if err := w.AddRecursive("./test"); err != nil {
		log.Fatalln(err)
	}

	// Start the watching process - it'll check for changes every 100ms.
	if err := w.Start(time.Millisecond * 100); err != nil {
		log.Fatalln(err)
	}
}

func syncEnvironments() {
	s3Files := listS3Files("my-go-box")
	systemFiles := getSystemFiles("test")
	s3FileNames := map[string]bool{}
	systemFileNames := map[string]bool{}
	for _, s3File := range s3Files.Contents {
		s3FileNames[*s3File.Key] = true
	}
	for _, systemFile := range systemFiles {
		systemFileNames[systemFile.Name()] = true
	}
	missingFromSystem := findFilesMissingFromSystem(s3FileNames, systemFileNames)
	missingFromS3 := findFilesMissingFromS3(s3FileNames, systemFileNames)

	log.Print(strconv.Itoa(len(missingFromSystem)) + " missing from your system")
	log.Print(strconv.Itoa(len(missingFromS3)) + " missing from your cloud")

	downloadMissingFiles(missingFromSystem)
	uploadMissingFiles(missingFromS3)
}

func listS3Files(bucket string) *s3.ListObjectsOutput {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-2"),
	})

	if err != nil {
		log.Fatalln(err)
		return nil
	}

	input := &s3.ListObjectsInput{
		Bucket: aws.String(bucket),
	}

	s3Service := s3.New(sess)
	files, s3Error := s3Service.ListObjects(input)
	if s3Error != nil {
		log.Fatalln(s3Error)
		return nil
	}

	return files
}

func getSystemFiles(path string) []os.FileInfo {

	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatalln(err)
		return nil
	}

	return files
}

func findFilesMissingFromSystem(s3FileNames, systemFileNames map[string]bool) []string {

	var missingFiles []string
	for name, _ := range s3FileNames {
		if !systemFileNames[name] {
			missingFiles = append(missingFiles, name)
		}
	}

	return missingFiles
}

func findFilesMissingFromS3(s3FileNames, systemFileNames map[string]bool) []string {

	var missingFiles []string
	for name, _ := range systemFileNames {
		if !s3FileNames[name] {
			missingFiles = append(missingFiles, name)
		}
	}

	return missingFiles
}

func downloadMissingFiles(missingFromSystem []string) {
	for _, missingFileName := range missingFromSystem {

		sess, err := session.NewSession(&aws.Config{
			Region: aws.String("us-east-2"),
		})

		if err != nil {
			log.Fatalln(err)
		}

		getInput := &s3.GetObjectInput{
			Bucket: aws.String("my-go-box"),
			Key:    aws.String(missingFileName),
		}

		s3Service := s3.New(sess)
		f, s3Error := s3Service.GetObject(getInput)
		if s3Error != nil {
			log.Fatalln(s3Error)
			return
		}

		writeFile, ioError := os.Create("test/" + missingFileName)
		if ioError != nil {
			log.Fatalln(ioError)
			return
		}

		bytes, _ := ioutil.ReadAll(f.Body)
		writeFile.Write(bytes)
	}
}

func uploadMissingFiles(missingFromS3 []string) {
	for _, missingFileName := range missingFromS3 {
		uploadFile(missingFileName)
	}
}

func uploadFile(fileName string) {
	f, fErr := os.Open("./test/" + fileName)
	if fErr != nil {
		log.Fatalln(fErr)
	}
	fileInfo, _ := f.Stat()
	buffer := make([]byte, fileInfo.Size())
	_, _ = f.Read(buffer)
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-2"),
	})
	if err != nil {
		log.Fatalln(err)
	}
	putInput := &s3.PutObjectInput{
		Bucket: aws.String("my-go-box"),
		Key:    aws.String(fileName),
		Body:   bytes.NewReader(buffer),
	}
	s3Service := s3.New(sess)
	_, s3Error := s3Service.PutObject(putInput)
	if s3Error != nil {
		log.Fatalln(s3Error)
	}
}

func deleteFileFromS3(fileName string) {

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-2"),
	})

	if err != nil {
		log.Fatalln(err)
	}

	deleteInput := &s3.DeleteObjectInput{
		Bucket: aws.String("my-go-box"),
		Key: aws.String(fileName),
	}

	s3Service := s3.New(sess)
	_, s3Error := s3Service.DeleteObject(deleteInput)
	if s3Error != nil {
		log.Fatalln(s3Error)
	}
}