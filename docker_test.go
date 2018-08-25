package main

import (
	"context"
	"errors"
	"io/ioutil"
	"reflect"
	"testing"

	"docker.io/go-docker/api/types"
	"docker.io/go-docker/api/types/container"
	"docker.io/go-docker/api/types/events"
	"docker.io/go-docker/api/types/network"
	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetOutput(ioutil.Discard)
}

type testClient struct{}

func (testClient) ContainerList(_ context.Context, _ types.ContainerListOptions) ([]types.Container, error) {
	return []types.Container{
		{
			ID: "111",
			Names: []string{
				"/someservice",
			},
		},
		{
			ID: "222",
			Names: []string{
				"/someproject_someservice_1",
			},
		},
		{
			ID: "333",
			Names: []string{
				"/someotherproject_someotherservice_1",
			},
		},
		{
			ID: "444",
			Names: []string{
				"/some_nonnetworked_service",
			},
		},
	}, nil
}

func (testClient) ContainerInspect(_ context.Context, ID string) (types.ContainerJSON, error) {
	switch ID {
	case "111":
		return types.ContainerJSON{
			ContainerJSONBase: &types.ContainerJSONBase{Name: "service1"},
			Config:            &container.Config{Labels: map[string]string{}},
			NetworkSettings: &types.NetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"bridge": {
						IPAddress: "1.2.3.4",
						Aliases: []string{
							"somealias",
						},
					},
				},
			},
		}, nil
	case "222":
		return types.ContainerJSON{
			ContainerJSONBase: &types.ContainerJSONBase{Name: "service2"},
			Config: &container.Config{Labels: map[string]string{
				"com.docker.compose.project": "someproject",
			}},
			NetworkSettings: &types.NetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"somenetwork": {
						IPAddress: "2.3.4.5",
						Aliases: []string{
							"somealias1",
							"nonuniquealias",
						},
					},
				},
			},
		}, nil
	case "333":
		return types.ContainerJSON{
			ContainerJSONBase: &types.ContainerJSONBase{Name: "service3"},
			Config: &container.Config{Labels: map[string]string{
				"com.docker.compose.project": "someotherproject",
			}},
			NetworkSettings: &types.NetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"someothernetwork": {
						IPAddress: "3.4.5.6",
						Aliases: []string{
							"someotheralias1",
							"nonuniquealias",
						},
					},
					"somesecondarynetwork": {
						IPAddress: "4.5.6.7",
						Aliases: []string{
							"somesecondaryalias1",
						},
					},
				},
			},
		}, nil
	case "444":
		return types.ContainerJSON{
			ContainerJSONBase: &types.ContainerJSONBase{Name: "service4"},
			Config:            &container.Config{Labels: map[string]string{}},
			NetworkSettings: &types.NetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"none": {},
				},
			},
		}, nil
	default:
		panic("whaaa?")
	}
}

func (testClient) Events(context.Context, types.EventsOptions) (<-chan events.Message, <-chan error) {
	return nil, nil
}

func (testClient) Ping(context.Context) (types.Ping, error) {
	return types.Ping{}, nil
}

type workingPinger struct{}

func (workingPinger) Ping(_ context.Context) (types.Ping, error) {
	return types.Ping{}, nil
}

// not safe to use same pinger in multiple parallel tests!
type delayedPinger struct{ counter, limit int }

func (wp *delayedPinger) Ping(_ context.Context) (types.Ping, error) {
	// some arbitrary number of failures
	if wp.counter >= wp.limit {
		return types.Ping{}, nil
	}
	wp.counter++
	return types.Ping{}, errors.New("not working yet")
}

func Test_getIPsToNames(t *testing.T) {
	type args struct {
		client dockerClienter
		id     string
	}
	tests := []struct {
		name    string
		args    args
		want    ipsToNamesMap
		wantErr bool
	}{
		{"simple query1", args{testClient{}, "111"}, ipsToNamesMap{
			"1.2.3.4": []string{
				"service1", "somealias",
			},
		}, false},
		{"query with aliases and projects", args{testClient{}, "222"}, ipsToNamesMap{
			"2.3.4.5": []string{
				"service2", "service2.somenetwork", "service2.someproject", "service2.someproject.somenetwork",
				"somealias1", "somealias1.somenetwork", "somealias1.someproject", "somealias1.someproject.somenetwork",
				"nonuniquealias", "nonuniquealias.somenetwork", "nonuniquealias.someproject", "nonuniquealias.someproject.somenetwork",
			},
		}, false},
		{"query with 2 networks", args{testClient{}, "333"}, ipsToNamesMap{
			"3.4.5.6": []string{
				"service3", "service3.someothernetwork", "service3.someotherproject", "service3.someotherproject.someothernetwork",
				"someotheralias1", "someotheralias1.someothernetwork", "someotheralias1.someotherproject", "someotheralias1.someotherproject.someothernetwork",
				"nonuniquealias", "nonuniquealias.someothernetwork", "nonuniquealias.someotherproject", "nonuniquealias.someotherproject.someothernetwork",
			},
			"4.5.6.7": []string{
				"service3", "service3.somesecondarynetwork", "service3.someotherproject", "service3.someotherproject.somesecondarynetwork",
				"somesecondaryalias1", "somesecondaryalias1.somesecondarynetwork", "somesecondaryalias1.someotherproject", "somesecondaryalias1.someotherproject.somesecondarynetwork",
			},
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getIPsToNames(tt.args.client, tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("getIPsToNames() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getIPsToNames():\n%v\nwant:\n%v", got, tt.want)
			}
		})
	}
}

func Test_getAllIPsToNames(t *testing.T) {
	type args struct {
		client dockerClienter
	}
	tests := []struct {
		name    string
		args    args
		want    ipsToNamesMap
		wantErr bool
	}{
		{"simple query1", args{testClient{}}, ipsToNamesMap{
			"1.2.3.4": []string{"service1", "somealias"},
			"2.3.4.5": []string{
				"service2", "service2.somenetwork", "service2.someproject", "service2.someproject.somenetwork",
				"somealias1", "somealias1.somenetwork", "somealias1.someproject", "somealias1.someproject.somenetwork",
				"nonuniquealias", "nonuniquealias.somenetwork", "nonuniquealias.someproject", "nonuniquealias.someproject.somenetwork",
			},
			"3.4.5.6": []string{
				"service3", "service3.someothernetwork", "service3.someotherproject", "service3.someotherproject.someothernetwork",
				"someotheralias1", "someotheralias1.someothernetwork", "someotheralias1.someotherproject", "someotheralias1.someotherproject.someothernetwork",
				"nonuniquealias", "nonuniquealias.someothernetwork", "nonuniquealias.someotherproject", "nonuniquealias.someotherproject.someothernetwork",
			},
			"4.5.6.7": []string{
				"service3", "service3.somesecondarynetwork", "service3.someotherproject", "service3.someotherproject.somesecondarynetwork",
				"somesecondaryalias1", "somesecondaryalias1.somesecondarynetwork", "somesecondaryalias1.someotherproject", "somesecondaryalias1.someotherproject.somesecondarynetwork",
			},
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getAllIPsToNames(tt.args.client)
			if (err != nil) != tt.wantErr {
				t.Errorf("getAllIPsToNames() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getAllIPsToNames() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_waitForConnection(t *testing.T) {
	type args struct {
		client dockerClientPinger
	}
	tests := []struct {
		name string
		args args
		long bool // whether this test is "long-running"
	}{
		{"connection working", args{workingPinger{}}, false},
		{"delayed connection", args{&delayedPinger{limit: 5}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.long && testing.Short() {
				t.Skip()
			}
			waitForConnection(tt.args.client)
		})
	}
}
