package config

// https://api.telemetry.confluent.cloud/docs#section/Object-Model/Metrics

type MetricModel struct {
	Name   string
	Labels []string
}

// TODO: Add ksql and schema_registry
var ObjectModel = map[string][]*MetricModel{
	"kafka": {
		{Name: "confluent_kafka_server_received_bytes", Labels: []string{"kafka_id", "topic"}},
		{Name: "confluent_kafka_server_sent_bytes", Labels: []string{"kafka_id", "topic"}},
		{Name: "confluent_kafka_server_received_records", Labels: []string{"kafka_id", "topic"}},
		{Name: "confluent_kafka_server_sent_records", Labels: []string{"kafka_id", "topic"}},
		{Name: "confluent_kafka_server_retained_bytes", Labels: []string{"kafka_id", "topic"}},
		{Name: "confluent_kafka_server_active_connection_count", Labels: []string{"kafka_id", "principal_id"}},
		{Name: "confluent_kafka_server_request_count", Labels: []string{"kafka_id", "principal_id", "type"}},
		{Name: "confluent_kafka_server_partition_count", Labels: []string{"kafka_id"}},
		{Name: "confluent_kafka_server_successful_authentication_count", Labels: []string{"kafka_id", "principal_id"}},
	},
	"connector": {
		{Name: "confluent_kafka_connect_sent_records", Labels: []string{"connector_id"}},
		{Name: "confluent_kafka_connect_received_records", Labels: []string{"connector_id"}},
		{Name: "confluent_kafka_connect_sent_bytes", Labels: []string{"connector_id"}},
		{Name: "confluent_kafka_connect_received_bytes", Labels: []string{"connector_id"}},
		{Name: "confluent_kafka_connect_dead_letter_queue_records", Labels: []string{"connector_id"}},
	},
}
