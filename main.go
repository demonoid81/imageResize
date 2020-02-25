package main

import (
	"bytes"
	"fmt"
	"github.com/disintegration/imaging"
	"github.com/streadway/amqp"
	"github.com/ugorji/go/codec"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"

	"os"
)


func toBin (height  int, width int, img image.Image) []byte {
	var pixels []byte
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			pixels = append(pixels, byte(r/257))
			pixels = append(pixels, byte(g/257))
			pixels = append(pixels, byte(b/257))
		}
	}
	return pixels
}


func main() {
	//fileToBeUploaded := "img.png"
	//file, err := os.Open(fileToBeUploaded)
	//if err != nil {
	//	fmt.Println(err)
	//	os.Exit(1)
	//}
	//
	//defer file.Close()
	//
	//fileInfo, _ := file.Stat()
	//var size int64 = fileInfo.Size()
	//imgBytes := make([]byte, size)
	//
	//// read file into bytes
	//buffer := bufio.NewReader(file)
	//_, err = buffer.Read(imgBytes)
	//
	//
	//
	//var data []byte
	//original := make(map[string]interface{})
	//
	////original["height"] = height
	////original["width"] = width
	//original["image"] = imgBytes
	//
	//
	//enc := codec.NewEncoderBytes(&data, new(codec.MsgpackHandle))
	//if err := enc.Encode(&original); err != nil {
	//	panic(err)
	//}
	//
	//fmt.Println(data[:40])

	host := os.Getenv("AMQP_HOST")
	port := os.Getenv("AMQP_PORT")
	user := os.Getenv("AMQP_USER")
	password := os.Getenv("AMQP_PASS")
	//
	var url string

	if user != "" || password != "" {
		url = "amqp://" + user + ":" + password + "@" + host + ":" + port
	} else {
		url = "amqp://" + host + ":" + port
	}
	//

	fmt.Println(url)

	connection, err := amqp.Dial(url)

	if err != nil {
		panic("could not establish connection with RabbitMQ:" + err.Error())
	}

	channel, err := connection.Channel()
	defer channel.Close()

	if err != nil {
		panic("could not open RabbitMQ channel:" + err.Error())
	}
	//
	//
	//queue1, err := channel.QueueDeclare("enface.image1", false, false, false, false, nil)
	//if err != nil {
	//	panic("could not open RabbitMQ channel:" + err.Error())
	//}
	//
	//err = channel.Publish("", queue1.Name, false, false, amqp.Publishing{
	//	Body: data,
	//})
	//
	//if err != nil {
	//	panic("could not open RabbitMQ channel:" + err.Error())
	//}

	defer connection.Close()

	queue, err := channel.QueueDeclare("enface.image1", false, false, false, false, nil)
	if err != nil {
		panic("could not open RabbitMQ channel:" + err.Error())
	}

	err = channel.Qos(1, 0, false)
	if err != nil {
		panic("could not open RabbitMQ channel:" + err.Error())
	}

	messageChannel, err := channel.Consume(
		queue.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		panic("could not open RabbitMQ channel:" + err.Error())
	}

	stopChan := make(chan bool)

	go func() {
		log.Printf("Consumer ready, PID: %d", os.Getpid())
		for d := range messageChannel {
			log.Println("Received a message")
			decoded := make(map[string]interface{})
			dec := codec.NewDecoderBytes(d.Body, new(codec.MsgpackHandle))
			if err := dec.Decode(&decoded); err != nil {
				panic(err)
			}

			imgBytes = decoded["image"].([]byte)

			img, _, err := image.Decode(bytes.NewReader(imgBytes))
			if err != nil {
				panic(err)
			}
			bounds := img.Bounds()

			width, height := bounds.Max.X, bounds.Max.Y

			originalPixels := toBin(height, width, img)

			var data []byte

			image480 := imaging.Fit(img, 480,360, imaging.Lanczos)

			resizePixels := toBin(360, 480, image480)

			// Save the resulting image as JPEG.
			//err = imaging.Save(image480, "example.jpg")
			//if err != nil {
			//	log.Fatalf("failed to save image: %v", err)
			//}

			decoded["height"] = height
			decoded["width"] = width
			decoded["original"] = originalPixels
			decoded["resize"] = resizePixels

			enc := codec.NewEncoderBytes(&data, new(codec.MsgpackHandle))
			if err := enc.Encode(&decoded); err != nil {
				panic(err)
			}

			queue, err := channel.QueueDeclare("enface.pps1", false, false, false, false, nil)
			if err != nil {
				panic("could not open RabbitMQ channel:" + err.Error())
			}

			err = channel.Publish("", queue.Name, false, false, amqp.Publishing{
				Body: data,
			})

			if err != nil {
				panic("could not open RabbitMQ channel:" + err.Error())
			}

			if err := d.Ack(false); err != nil {
				log.Printf("Error acknowledging message : %s", err)
			} else {
				log.Printf("Acknowledged message")
			}

		}
	}()

	// Stop for program termination
	<-stopChan

}
