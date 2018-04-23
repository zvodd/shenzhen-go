// Copyright 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"log"

	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/google/shenzhen-go/dev/model"
	pb "github.com/google/shenzhen-go/dev/proto/go"
)

func (c *server) CreateChannel(ctx context.Context, req *pb.CreateChannelRequest) (*pb.Empty, error) {
	log.Printf("api: CreateChannel(%s)", proto.MarshalTextString(req))
	g, err := c.lookupGraph(req.Graph)
	if err != nil {
		return &pb.Empty{}, err
	}
	g.Lock()
	defer g.Unlock()

	np := make(map[model.NodePin]struct{}, len(req.Pins))
	for _, x := range req.Pins {
		n, err := g.lookupNode(x.Node)
		if err != nil {
			return &pb.Empty{}, err
		}
		if co := n.Connections[x.Pin]; co != "nil" {
			return &pb.Empty{}, status.Errorf(codes.FailedPrecondition, "node %q pin %q either does not exist or is already connected (%q)", x.Node, x.Pin, co)
		}
	}
	// TODO: better validation
	if req.Name == "nil" {
		return &pb.Empty{}, status.Errorf(codes.InvalidArgument, "channels may not be named %q", req.Name)
	}
	if _, found := g.Channels[req.Name]; found {
		return &pb.Empty{}, status.Errorf(codes.FailedPrecondition, "channel %q already exists", req.Name)
	}
	// TODO: validate the name isn't silly, the type isn't silly...
	g.Channels[req.Name] = &model.Channel{
		Name:      req.Name,
		Type:      req.Type,
		Anonymous: req.Anon,
		Capacity:  int(req.Cap),
		Pins:      np,
	}
	for _, x := range req.Pins {
		g.Nodes[x.Node].Connections[x.Pin] = req.Name
	}
	return &pb.Empty{}, nil
}

func (c *server) CreateNode(ctx context.Context, req *pb.CreateNodeRequest) (*pb.Empty, error) {
	log.Printf("api: CreateNode(%s)", proto.MarshalTextString(req))
	g, err := c.lookupGraph(req.Graph)
	if err != nil {
		return &pb.Empty{}, err
	}
	g.Lock()
	defer g.Unlock()

	if _, found := g.Nodes[req.Props.Name]; found {
		return &pb.Empty{}, status.Errorf(codes.FailedPrecondition, "node %q already exists", req.Props.Name)
	}
	p, err := (&model.PartJSON{
		Part: req.Props.PartCfg,
		Type: req.Props.PartType,
	}).Unmarshal()
	if err != nil {
		return &pb.Empty{}, status.Errorf(codes.FailedPrecondition, "part unmarshal: %v", err)
	}
	n := &model.Node{
		Name:         req.Props.Name,
		Multiplicity: uint(req.Props.Multiplicity),
		Enabled:      req.Props.Enabled,
		Wait:         req.Props.Wait,
		Part:         p,
		X:            req.Props.X,
		Y:            req.Props.Y,
		Connections:  make(map[string]string),
	}
	for _, d := range p.Pins() {
		n.Connections[d.Name] = "nil"
	}
	g.Nodes[req.Props.Name] = n
	return &pb.Empty{}, nil
}

func (c *server) ConnectPin(ctx context.Context, req *pb.ConnectPinRequest) (*pb.Empty, error) {
	log.Printf("api: ConnectPin(%s)", proto.MarshalTextString(req))
	g, err := c.lookupGraph(req.Graph)
	if err != nil {
		return &pb.Empty{}, err
	}
	g.Lock()
	defer g.Unlock()

	n, err := g.lookupNode(req.Node)
	if err != nil {
		return &pb.Empty{}, err
	}
	ch, err := g.lookupChannel(req.Channel)
	if err != nil {
		return &pb.Empty{}, err
	}
	if _, found := n.Pins()[req.Pin]; !found {
		return &pb.Empty{}, status.Errorf(codes.NotFound, "no pin %q on node %q", req.Pin, req.Node)
	}
	// TODO: Report type mismatches later on.
	n.Connections[req.Pin] = req.Channel
	ch.Pins[model.NodePin{Node: req.Node, Pin: req.Pin}] = struct{}{}
	return &pb.Empty{}, nil
}

func (c *server) DeleteChannel(ctx context.Context, req *pb.DeleteChannelRequest) (*pb.Empty, error) {
	log.Printf("api: DeleteChannel(%s)", proto.MarshalTextString(req))
	g, err := c.lookupGraph(req.Graph)
	if err != nil {
		return &pb.Empty{}, err
	}
	g.Lock()
	defer g.Unlock()
	ch, err := g.lookupChannel(req.Channel)
	if err != nil {
		return &pb.Empty{}, err
	}
	g.DeleteChannel(ch)
	return &pb.Empty{}, nil
}

func (c *server) DeleteNode(ctx context.Context, req *pb.DeleteNodeRequest) (*pb.Empty, error) {
	log.Printf("api: DeleteNode(%s)", proto.MarshalTextString(req))
	g, err := c.lookupGraph(req.Graph)
	if err != nil {
		return &pb.Empty{}, err
	}
	g.Lock()
	defer g.Unlock()
	n, err := g.lookupNode(req.Node)
	if err != nil {
		return &pb.Empty{}, err
	}
	delete(g.Nodes, req.Node)
	// Also clean up channel connections.
	for p, cn := range n.Connections {
		if cn == "nil" {
			continue
		}
		n.Connections[p] = "nil"
		ch := g.Channels[cn]
		if ch == nil {
			continue
		}
		delete(ch.Pins, model.NodePin{Node: req.Node, Pin: p})
		if len(ch.Pins) < 2 {
			g.DeleteChannel(ch)
		}
	}
	return &pb.Empty{}, nil
}

func (c *server) DisconnectPin(ctx context.Context, req *pb.DisconnectPinRequest) (*pb.Empty, error) {
	log.Printf("api: DisconnectPin(%s)", proto.MarshalTextString(req))
	g, err := c.lookupGraph(req.Graph)
	if err != nil {
		return &pb.Empty{}, err
	}
	g.Lock()
	defer g.Unlock()
	n, err := g.lookupNode(req.Node)
	if err != nil {
		return &pb.Empty{}, err
	}
	cn, found := n.Connections[req.Pin]
	if !found {
		return &pb.Empty{}, status.Errorf(codes.NotFound, "no pin %q on node %q", req.Pin, req.Node)
	}
	n.Connections[req.Pin] = "nil"
	// Clean up channel if unneccessary while we're at it.
	if ch := g.Channels[cn]; ch != nil {
		delete(ch.Pins, model.NodePin{Node: req.Node, Pin: req.Pin})
		if len(ch.Pins) < 2 {
			g.DeleteChannel(ch)
		}
	}
	return &pb.Empty{}, nil
}

func (c *server) Save(ctx context.Context, req *pb.SaveRequest) (*pb.Empty, error) {
	log.Printf("api: Save(%s)", proto.MarshalTextString(req))
	g, err := c.lookupGraph(req.Graph)
	if err != nil {
		return &pb.Empty{}, err
	}
	g.Lock()
	defer g.Unlock()
	return &pb.Empty{}, SaveJSONFile(g.Graph)
}

func (c *server) SetGraphProperties(ctx context.Context, req *pb.SetGraphPropertiesRequest) (*pb.Empty, error) {
	log.Printf("api: SetGraphProperties(%s)", proto.MarshalTextString(req))
	g, err := c.lookupGraph(req.Graph)
	if err != nil {
		return &pb.Empty{}, err
	}
	g.Name = req.Name
	g.PackagePath = req.PackagePath
	g.IsCommand = req.IsCommand
	return &pb.Empty{}, nil
}

func (c *server) SetNodeProperties(ctx context.Context, req *pb.SetNodePropertiesRequest) (*pb.Empty, error) {
	log.Printf("api: SetNodeProperties(%s)", proto.MarshalTextString(req))
	g, err := c.lookupGraph(req.Graph)
	if err != nil {
		return &pb.Empty{}, err
	}
	g.Lock()
	defer g.Unlock()
	n, err := g.lookupNode(req.Node)
	if err != nil {
		return &pb.Empty{}, err
	}

	// TODO: refactor; this is extremely similar to the file unmarshaling code...
	p, err := (&model.PartJSON{
		Part: req.Props.PartCfg,
		Type: req.Props.PartType,
	}).Unmarshal()
	if err != nil {
		return &pb.Empty{}, status.Errorf(codes.FailedPrecondition, "part unmarshal: %v", err)
	}
	if n.Name != req.Props.Name {
		if _, exists := g.Nodes[req.Props.Name]; exists {
			return &pb.Empty{}, status.Errorf(codes.FailedPrecondition, "node %q already exists", req.Props.Name)
		}
		delete(g.Nodes, n.Name)
		n.Name = req.Props.Name
		g.Nodes[n.Name] = n
	}
	n.Multiplicity = uint(req.Props.Multiplicity)
	n.Enabled = req.Props.Enabled
	n.Wait = req.Props.Wait
	n.Part = p
	n.X, n.Y = req.Props.X, req.Props.Y
	n.RefreshConnections()
	g.RefreshChannelsPins() // Changing the part might have changed available pins.
	return &pb.Empty{}, nil
}

func (c *server) SetPosition(ctx context.Context, req *pb.SetPositionRequest) (*pb.Empty, error) {
	log.Printf("api: SetPosition(%s)", proto.MarshalTextString(req))
	g, err := c.lookupGraph(req.Graph)
	if err != nil {
		return &pb.Empty{}, err
	}
	g.Lock()
	defer g.Unlock()
	n, err := g.lookupNode(req.Node)
	if err != nil {
		return &pb.Empty{}, err
	}
	n.X, n.Y = req.X, req.Y
	return &pb.Empty{}, nil
}
