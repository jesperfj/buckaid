package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

var (
	logger = log.New(os.Stderr, "[Web] ", log.Ldate|log.Ltime|log.Lshortfile)
)

func getRequiredenv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		logger.Fatal(key, " must be set")
	}
	return val
}

func main() {
	port := getRequiredenv("PORT")
	bucketName := getRequiredenv("BYODEMO_BUCKET_NAME")
	awsAccessKeyId := getRequiredenv("BYODEMO_AWS_ACCESS_KEY_ID")
	awsSecretAccessKey := getRequiredenv("BYODEMO_AWS_SECRET_ACCESS_KEY")

	sess, _ := session.NewSession(&aws.Config{
		Region:      aws.String("us-east-1"),
		Credentials: credentials.NewStaticCredentials(awsAccessKeyId, awsSecretAccessKey, ""),
	})
	s3svc := s3.New(sess)

	// General routing setup
	router := gin.New()
	router.Use(gin.Logger())
	router.LoadHTMLGlob("templates/*.tmpl.html")

	// Renders the HTML interface
	router.GET("/", func(c *gin.Context) {
		res, _ := s3svc.ListObjects(&s3.ListObjectsInput{Bucket: &bucketName})
		c.HTML(http.StatusOK, "index.tmpl.html", gin.H{"files": res.Contents, "host": c.Request.Host})
	})

	// Responds with list of files in text format (meant for curl)
	router.GET("/files", func(c *gin.Context) {
		res, _ := s3svc.ListObjects(&s3.ListObjectsInput{Bucket: &bucketName})
		for _, o := range res.Contents {
			c.Writer.Write([]byte(*o.Key))
			c.Writer.Write([]byte("\n"))
			logger.Print(*o.Key)
		}
		c.Status(200)
	})

	// For uploading a file with curl, use:
	// curl -L http://myapp.mydomain/put/filenameinbucket --upload-file filenameondisk
	router.PUT("/put/:key", func(c *gin.Context) {
		key := c.Param("key")
		req, _ := s3svc.PutObjectRequest(&s3.PutObjectInput{
			Bucket: &bucketName,
			Key:    &key,
		})
		str, _ := req.Presign(2 * time.Minute)
		c.Redirect(302, str)
	})

	// For downloading a file with curl, use:
	// curl -L -o filenameondisk http://myapp.mydomain/get/filenameinbucket
	router.GET("/get/:key", func(c *gin.Context) {
		key := c.Param("key")
		req, _ := s3svc.GetObjectRequest(&s3.GetObjectInput{
			Bucket: &bucketName,
			Key:    &key,
		})
		str, _ := req.Presign(2 * time.Minute)
		c.Redirect(302, str)
	})

	router.Run(":" + port)
}
