package azblob_test

import (
	"context"
	"github.com/Azure/azure-storage-blob-go/2017-07-29/azblob"
	chk "gopkg.in/check.v1"
	"io/ioutil"
	"os"
)

// create a test file
func generateFile(fileName string, fileSize int) []byte {
	// generate random data
	_, bigBuff := getRandomDataAndReader(fileSize)

	// write to file and return the data
	ioutil.WriteFile(fileName, bigBuff, 0666)
	return bigBuff
}

func performUploadStreamToBlockBlobTest(c *chk.C, blobSize, bufferSize, maxBuffers int) {
	// Set up test container
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)

	// Set up test blob
	blobURL, _ := getBlockBlobURL(c, containerURL)

	// Create a some data to test the upload stream
	blobContentReader, blobData := getRandomDataAndReader(blobSize)

	// Perform UploadStreamToBlockBlob
	uploadResp, err := azblob.UploadStreamToBlockBlob(ctx, blobContentReader, blobURL,
		azblob.UploadStreamToBlockBlobOptions{BufferSize: bufferSize, MaxBuffers: maxBuffers})

	// Assert that upload was successful
	c.Assert(err, chk.Equals, nil)
	c.Assert(uploadResp.Response().StatusCode, chk.Equals, 201)

	// Download the blob to verify
	downloadResponse, err := blobURL.Download(ctx, 0, 0, azblob.BlobAccessConditions{}, false)
	c.Assert(err, chk.IsNil)

	// Assert that the content is correct
	actualBlobData, err := ioutil.ReadAll(downloadResponse.Response().Body)
	c.Assert(err, chk.IsNil)
	c.Assert(len(actualBlobData), chk.Equals, blobSize)
	c.Assert(actualBlobData, chk.DeepEquals, blobData)
}

func (s *aztestsSuite) TestUploadStreamToBlockBlobInChunks(c *chk.C) {
	blobSize := 8 * 1024
	bufferSize := 1024
	maxBuffers := 3
	performUploadStreamToBlockBlobTest(c, blobSize, bufferSize, maxBuffers)
}

func (s *aztestsSuite) TestUploadStreamToBlockBlobSingleBuffer(c *chk.C) {
	blobSize := 8 * 1024
	bufferSize := 1024
	maxBuffers := 1
	performUploadStreamToBlockBlobTest(c, blobSize, bufferSize, maxBuffers)
}

func (s *aztestsSuite) TestUploadStreamToBlockBlobSingleIO(c *chk.C) {
	blobSize := 1024
	bufferSize := 8 * 1024
	maxBuffers := 3
	performUploadStreamToBlockBlobTest(c, blobSize, bufferSize, maxBuffers)
}

// TODO currently failing due to empty body
func (s *aztestsSuite) TestUploadStreamToBlockBlobEmpty(c *chk.C) {
	blobSize := 0
	bufferSize := 8 * 1024
	maxBuffers := 3
	performUploadStreamToBlockBlobTest(c, blobSize, bufferSize, maxBuffers)
}

func performUploadAndDownloadFileTest(c *chk.C, fileSize, blockSize, parallelism int) {
	// Set up file to upload
	fileName := "BigFile.bin"
	fileData := generateFile(fileName, fileSize)

	// Open the file to upload
	file, err := os.Open(fileName)
	c.Assert(err, chk.Equals, nil)
	defer file.Close()
	defer os.Remove(fileName)

	// Set up test container
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)

	// Set up test blob
	blockBlobURL, _ := getBlockBlobURL(c, containerURL)

	// Upload the file to a block blob
	response, err := azblob.UploadFileToBlockBlob(context.Background(), file, blockBlobURL,
		azblob.UploadToBlockBlobOptions{
			BlockSize:   int64(blockSize),
			Parallelism: uint16(parallelism),
			// If Progress is non-nil, this function is called periodically as bytes are uploaded.
			Progress: func(bytesTransferred int64) {
				c.Assert(bytesTransferred > 0 && bytesTransferred <= int64(fileSize), chk.Equals, true)
			},
		})
	c.Assert(err, chk.Equals, nil)
	c.Assert(response.Response().StatusCode, chk.Equals, 201)

	// Set up file to download the blob to
	destFileName := "BigFile-downloaded.bin"
	destFile, err := os.Create(destFileName)
	c.Assert(err, chk.Equals, nil)
	defer destFile.Close()
	defer os.Remove(destFileName)

	// Perform download
	err = azblob.DownloadBlobToFile(context.Background(), blockBlobURL.BlobURL, azblob.BlobAccessConditions{}, destFile,
		azblob.DownloadFromBlobOptions{
			BlockSize:   int64(blockSize),
			Parallelism: uint16(parallelism),
			// If Progress is non-nil, this function is called periodically as bytes are uploaded.
			Progress: func(bytesTransferred int64) {
				c.Assert(bytesTransferred > 0 && bytesTransferred <= int64(fileSize), chk.Equals, true)
			},})

	// Assert download was successful
	c.Assert(err, chk.Equals, nil)

	// Assert downloaded data is consistent
	destBuffer := make([]byte, fileSize)
	n, err := destFile.Read(destBuffer)
	c.Assert(err, chk.Equals, nil)
	c.Assert(n, chk.Equals, fileSize)
	c.Assert(destBuffer, chk.DeepEquals, fileData)
}

func (s *aztestsSuite) TestUploadAndDownloadFileInChunks(c *chk.C) {
	fileSize := 8 * 1024
	blockSize := 1024
	parallelism := 3
	performUploadAndDownloadFileTest(c, fileSize, blockSize, parallelism)
}

func (s *aztestsSuite) TestUploadAndDownloadFileSingleIO(c *chk.C) {
	fileSize := 1024
	blockSize := 2048
	parallelism := 3
	performUploadAndDownloadFileTest(c, fileSize, blockSize, parallelism)
}

func (s *aztestsSuite) TestUploadAndDownloadFileSingleRoutine(c *chk.C) {
	fileSize := 8 * 1024
	blockSize := 1024
	parallelism := 1
	performUploadAndDownloadFileTest(c, fileSize, blockSize, parallelism)
}

// TODO currently failing due to empty body
func (s *aztestsSuite) TestUploadAndDownloadFileEmpty(c *chk.C) {
	fileSize := 0
	blockSize := 1024
	parallelism := 1
	performUploadAndDownloadFileTest(c, fileSize, blockSize, parallelism)
}

func performUploadAndDownloadBufferTest(c *chk.C, bufferSize, blockSize, parallelism int) {
	// Set up buffer to upload
	_, bytesToUpload := getRandomDataAndReader(bufferSize)

	// Set up test container
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)

	// Set up test blob
	blockBlobURL, _ := getBlockBlobURL(c, containerURL)

	// Pass the Context, stream, stream size, block blob URL, and options to StreamToBlockBlob
	response, err := azblob.UploadBufferToBlockBlob(context.Background(), bytesToUpload, blockBlobURL,
		azblob.UploadToBlockBlobOptions{
			BlockSize:   int64(blockSize),
			Parallelism: uint16(parallelism),
			// If Progress is non-nil, this function is called periodically as bytes are uploaded.
			Progress: func(bytesTransferred int64) {
				c.Assert(bytesTransferred > 0 && bytesTransferred <= int64(bufferSize), chk.Equals, true)
			},
		})
	c.Assert(err, chk.Equals, nil)
	c.Assert(response.Response().StatusCode, chk.Equals, 201)

	// Set up buffer to download the blob to
	destBuffer := make([]byte, bufferSize)

	// Download the blob to a buffer
	err = azblob.DownloadBlobToBuffer(context.Background(), blockBlobURL.BlobURL, 0, int64(bufferSize),
		azblob.BlobAccessConditions{}, destBuffer, azblob.DownloadFromBlobOptions{
			BlockSize:   int64(blockSize),
			Parallelism: uint16(parallelism),
			// If Progress is non-nil, this function is called periodically as bytes are uploaded.
			Progress: func(bytesTransferred int64) {
				c.Assert(bytesTransferred > 0 && bytesTransferred <= int64(bufferSize), chk.Equals, true)
			},
		})

	c.Assert(err, chk.Equals, nil)
	c.Assert(destBuffer, chk.DeepEquals, bytesToUpload)
}

func (s *aztestsSuite) TestUploadAndDownloadBufferInChunks(c *chk.C) {
	bufferSize := 8 * 1024
	blockSize := 1024
	parallelism := 3
	performUploadAndDownloadBufferTest(c, bufferSize, blockSize, parallelism)
}

func (s *aztestsSuite) TestUploadAndDownloadBufferSingleIO(c *chk.C) {
	bufferSize := 1024
	blockSize := 8 * 1024
	parallelism := 3
	performUploadAndDownloadBufferTest(c, bufferSize, blockSize, parallelism)
}

func (s *aztestsSuite) TestUploadAndDownloadBufferSingleRoutine(c *chk.C) {
	bufferSize := 8 * 1024
	blockSize := 1024
	parallelism := 1
	performUploadAndDownloadBufferTest(c, bufferSize, blockSize, parallelism)
}

// TODO currently failing due to empty body
func (s *aztestsSuite) TestUploadAndDownloadBufferEmpty(c *chk.C) {
	bufferSize := 8 * 1024
	blockSize := 1024
	parallelism := 3
	performUploadAndDownloadBufferTest(c, bufferSize, blockSize, parallelism)
}