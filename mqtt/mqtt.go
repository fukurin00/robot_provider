package mqtt

import (
	"fmt"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Connect to Mqtt Broker
func ConnectMqttBroker(
	srv string,
	port int,
	connectHandler func(client mqtt.Client),
	connectLostHandler func(client mqtt.Client, err error),
	messagePubHandler func(client mqtt.Client, msg mqtt.Message),
) *mqtt.Client {
	opts := mqtt.NewClientOptions()
	opts.SetAutoReconnect(true)
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", srv, port))
	// opts.SetClientID("go_mqtt_client")
	// opts.SetUsername("emqx")
	// opts.SetPassword("public")
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Panic(token.Error())
	}
	return &client
}
