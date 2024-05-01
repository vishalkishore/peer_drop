package transport

import (
    "bufio"
    "encoding/json"
    "os"
    "io"
    "fmt"
    CONSTANT "ftp_server/src/constant"
    "time"
    pb "github.com/cheggaaa/pb/v3"
)

type FileMetadata struct {
    Name string `json:"name"`
    Size int64  `json:"size"`
}

func SendFile(writer *bufio.Writer, file *os.File) error {
    // Get the file size
    fileInfo, err := file.Stat()
    if err != nil {
        return err
    }

    metadata := FileMetadata{
        Name: fileInfo.Name(),
        Size: fileInfo.Size(),
    }

    // Serialize the metadata into a JSON string
    metadataJson, err := json.Marshal(metadata)
    if err != nil {
        return err
    }

    // Send the metadata to the receiver
    _, err = writer.WriteString(string(metadataJson) + "\n")
    if err != nil {
        return err
    }
    err = writer.Flush()
    if err != nil {
        return err
    }

    // // Create a new progress bar
    // bar := pb.StartNew(int(metadata.Size))

    bar := pb.New64(metadata.Size)
    bar.Set(pb.Bytes, true) // Set the units to bytes
    bar.SetRefreshRate(time.Millisecond * 10)
    bar.Start()

    buf := make([]byte, CONSTANT.CHUNKSIZE)
    for {
        n, err := file.Read(buf)
        if err != nil {
            if err == io.EOF {
                fmt.Println("End of file reached")
                break
            }
            return err
        }

        _, err = writer.Write(buf[:n])
        if err != nil {
            return err
        }
        err = writer.Flush()
        if err != nil {
            return err
        }

        // Update the progress bar
        bar.Add(n)
    }

    // Finish the progress bar
    bar.Finish()

    return nil
}

func SendMessage(writer *bufio.Writer, message string) error {
	_, err := writer.WriteString(message)
	if err != nil {
		return err
	}
	return writer.Flush()
}