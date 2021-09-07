package exporter

import (
	"testing"

	"github.com/Azure/aks-periscope/pkg/collector"
	"github.com/Azure/aks-periscope/pkg/interfaces"
)

func TestZip(t *testing.T) {
	tests := []struct {
		name    string
		want    int
		wantErr bool
	}{
		{
			name:    "get zip data",
			want:    1,
			wantErr: false,
		},
	}

	c := collector.NewNetworkOutboundCollector()
	dataProducer := []interfaces.DataProducer{}
	listDataProducer := append(dataProducer, c)

	buf, err := Zip(listDataProducer)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if (err != nil) != tt.wantErr {
				t.Errorf("Collect() error = %v, wantErr %v", err, tt.wantErr)
			}

			bytes := buf.Bytes()
			str := buf.String()
			if buf.Len() != len(bytes) {
				t.Errorf("Buf.Len() == %d, len(buf.Bytes()) == %d", buf.Len(), len(bytes))
			}

			if buf.Len() != len(str) {
				t.Errorf("Buf.Len() == %d, len(buf.String()) == %d", buf.Len(), len(str))
			}
		})
	}
}
