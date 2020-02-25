package main

import (
	"bytes"
	"fmt"
    "github.com/pixiv/go-libjpeg/jpeg"
	//"github.com/disintegration/imaging"
	"gopkg.in/h2non/bimg.v1"
	"github.com/streadway/amqp"
	"github.com/ugorji/go/codec"
	"image"
	//_ "image/jpeg"
	//_ "image/png"
	"log"

	"os"
	"time"
)


func toBin (height  int, width int, img image.Image) []byte {
	start := time.Now()
	var pixels []byte
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			pixels = append(pixels, byte(r/257))
			pixels = append(pixels, byte(g/257))
			pixels = append(pixels, byte(b/257))
		}
	}
	elapsed := time.Since(start)
	log.Printf("transform took %s", elapsed)
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
		"ReplyToConsumer",
		true,
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
			SS := time.Now()
			log.Println("Received a message")
			decoded := make(map[string]interface{})
			dec := codec.NewDecoderBytes(d.Body, new(codec.MsgpackHandle))
			if err := dec.Decode(&decoded); err != nil {
				panic(err)
			}

			imgBytes := decoded["image"].([]byte)

			//elapsed := time.Since(start)
			//log.Printf("image decoded took %s", elapsed)
			start := time.Now()

			imgOld, err := jpeg.DecodeIntoRGB(bytes.NewReader(imgBytes), &jpeg.DecoderOptions{})
			if err != nil {
				panic(err)
			}
			bounds := imgOld.Bounds()

			//
			width, height := bounds.Max.X, bounds.Max.Y
			fmt.Println(width)
			fmt.Println(height)
			////
			////originalPixels := toBin(height, width, img)
			////
			var data []byte
			//
			elapsed := time.Since(start)
			log.Printf("image ready took %s", elapsed)
			start = time.Now()
			//
			//image480 := imaging.Fit(img, 480,360, imaging.Lanczos)

			image480, err := bimg.NewImage(imgBytes).Resize(480, 360)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}

			img, err := jpeg.DecodeIntoRGB(bytes.NewReader(image480), &jpeg.DecoderOptions{})
			if err != nil {
				panic(err)
			}
			bounds = img.Bounds()

			//
			width, height = bounds.Max.X, bounds.Max.Y
			fmt.Println(width)
			fmt.Println(height)
			//bounds = image480.Bounds()
			//
			////
			//width, height = bounds.Max.X, bounds.Max.Y
			//
			//fmt.Println(width)
			//fmt.Println(height)
			//
			//resizePixels := toBin(360, 480, image480)
			//
			elapsed = time.Since(start)
			log.Printf("image transform took %s", elapsed)
			//start = time.Now()

			// Save the resulting image as JPEG.
			//err = imaging.Save(image480, "example.jpg")
			//if err != nil {
			//	log.Fatalf("failed to save image: %v", err)
			//}

			decoded["height"] = height
			decoded["width"] = width
			decoded["original"] = &img
			decoded["resize"] = &image480

			enc := codec.NewEncoderBytes(&data, new(codec.MsgpackHandle))
			if err := enc.Encode(&decoded); err != nil {
				panic(err)
			}

			//queue, err := channel.QueueDeclare(msg.ReplyTo, false, false, false, false, nil)
			//if err != nil {
			//	panic("could not open RabbitMQ channel:" + err.Error())
			//}

			err = channel.Publish("", d.ReplyTo, false, false, amqp.Publishing{
				CorrelationId: d.CorrelationId,
				Body: data,
			})

			if err != nil {
				panic("could not open RabbitMQ channel:" + err.Error())
			}

			//if err := d.Ack(false); err != nil {
			//	log.Printf("Error acknowledging message : %s", err)
			//} else {
			//	log.Printf("Acknowledged message")
			//}
			elapsed = time.Since(SS)
			log.Printf("Binomial took %s", elapsed)

		}
	}()

	// Stop for program termination
	<-stopChan

}
