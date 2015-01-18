// Copyright (C) 2014 The Syncthing Authors.
//
// This program is free software: you can redistribute it and/or modify it
// under the terms of the GNU General Public License as published by the Free
// Software Foundation, either version 3 of the License, or (at your option)
// any later version.
//
// This program is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or
// FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for
// more details.
//
// You should have received a copy of the GNU General Public License along
// with this program. If not, see <http://www.gnu.org/licenses/>.

package model

import (
	"bytes"
	"testing"
	"time"

	"github.com/syncthing/protocol"
	"github.com/syncthing/syncthing/internal/config"
	"github.com/syncthing/syncthing/internal/scanner"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/storage"
)

var files []protocol.FileInfo = []protocol.FileInfo{
	{
		Name:    "file1",
		Blocks:  []protocol.BlockInfo{blocks[0], blocks[1], blocks[2]},
		Version: 100,
	},
	{
		Name:    "file2",
		Blocks:  []protocol.BlockInfo{blocks[3], blocks[4], blocks[5]},
		Version: 101,
	},
	{
		Name:   "file3",
		Blocks: []protocol.BlockInfo{blocks[6], blocks[7], blocks[8]},
	},
}

func TestTempIndex(t *testing.T) {
	i := newTempIndex()
	i.Update(device1, "x", files)
	i.Update(device2, "x", files[1:])

	if len(i.Lookup("x", "file1", blocks[0].Hash)) != 1 {
		t.Error("Invalid number of devices")
	}

	if len(i.Lookup("x", "file2", blocks[4].Hash)) != 2 {
		t.Error("Invalid number of devices")
	}

	if len(i.Lookup("x", "file3", blocks[6].Hash)) != 2 {
		t.Error("Invalid number of devices")
	}

	if len(i.Lookup("x", "file3", blocks[0].Hash)) != 0 {
		t.Error("Invalid number of devices")
	}

	i.Update(device2, "x", files[1:2])

	if len(i.Lookup("x", "file1", blocks[0].Hash)) != 1 {
		t.Error("Invalid number of devices")
	}

	if len(i.Lookup("x", "file2", blocks[4].Hash)) != 2 {
		t.Error("Invalid number of devices")
	}

	if len(i.Lookup("x", "file3", blocks[6].Hash)) != 1 {
		t.Error("Invalid number of devices")
	}

	i.Update(device2, "x", nil)

	if len(i.Lookup("x", "file1", blocks[0].Hash)) != 1 {
		t.Error("Invalid number of devices")
	}

	if len(i.Lookup("x", "file2", blocks[4].Hash)) != 1 {
		t.Error("Invalid number of devices")
	}

	if len(i.Lookup("x", "file3", blocks[6].Hash)) != 1 {
		t.Error("Invalid number of devices")
	}

	i.Update(device1, "x", nil)

	if len(i.Lookup("x", "file1", blocks[0].Hash)) != 0 {
		t.Error("Invalid number of devices")
	}

	if len(i.Lookup("x", "file2", blocks[4].Hash)) != 0 {
		t.Error("Invalid number of devices")
	}

	if len(i.Lookup("x", "file3", blocks[6].Hash)) != 0 {
		t.Error("Invalid number of devices")
	}

}

func TestModelTempIndex(t *testing.T) {
	db, _ := leveldb.Open(storage.NewMemStorage(), nil)
	m := NewModel(config.Wrap("/tmp/test", config.Configuration{Devices: []config.DeviceConfiguration{{DeviceID: device1}, {DeviceID: device2}}}), "device", "syncthing", "dev", db)
	m.AddFolder(config.FolderConfiguration{ID: "x", Path: "testdata", Devices: []config.FolderDeviceConfiguration{{DeviceID: device1}, {DeviceID: device2}}})

	fc1 := &FakeConnection{
		id: device1,
	}
	fc2 := &FakeConnection{
		id: device2,
	}

	m.AddConnection(fc1, fc1)
	m.AddConnection(fc2, fc2)

	m.IndexUpdate(device1, "x", files, protocol.FlagIndexTemporary, nil)

	devs := m.availability("x", "file1", blocks[0].Hash)
	if len(devs) != 1 || devs[device1] != protocol.FlagRequestTemporary {
		t.Errorf("Wrong count or flag")
	}

	devs = m.availability("x", "file2", blocks[4].Hash)
	if len(devs) != 1 || devs[device1] != protocol.FlagRequestTemporary {
		t.Errorf("Wrong count or flag")
	}

	devs = m.availability("x", "file3", blocks[6].Hash)
	if len(devs) != 1 || devs[device1] != protocol.FlagRequestTemporary {
		t.Errorf("Wrong count or flag")
	}

	m.IndexUpdate(device2, "x", files, protocol.FlagIndexTemporary, nil)

	devs = m.availability("x", "file1", blocks[0].Hash)
	if len(devs) != 2 || devs[device1] != protocol.FlagRequestTemporary || devs[device2] != protocol.FlagRequestTemporary {
		t.Errorf("Wrong count or flag")
	}

	devs = m.availability("x", "file2", blocks[4].Hash)
	if len(devs) != 2 || devs[device1] != protocol.FlagRequestTemporary || devs[device2] != protocol.FlagRequestTemporary {
		t.Errorf("Wrong count or flag")
	}

	devs = m.availability("x", "file3", blocks[6].Hash)
	if len(devs) != 2 || devs[device1] != protocol.FlagRequestTemporary || devs[device2] != protocol.FlagRequestTemporary {
		t.Errorf("Wrong count or flag")
	}

	m.IndexUpdate(device1, "x", files[1:], protocol.FlagIndexTemporary, nil)

	devs = m.availability("x", "file1", blocks[0].Hash)
	if len(devs) != 1 || devs[device2] != protocol.FlagRequestTemporary {
		t.Errorf("Wrong count or flag")
	}

	devs = m.availability("x", "file2", blocks[4].Hash)
	if len(devs) != 2 || devs[device1] != protocol.FlagRequestTemporary || devs[device2] != protocol.FlagRequestTemporary {
		t.Errorf("Wrong count or flag")
	}

	devs = m.availability("x", "file3", blocks[6].Hash)
	if len(devs) != 2 || devs[device1] != protocol.FlagRequestTemporary || devs[device2] != protocol.FlagRequestTemporary {
		t.Errorf("Wrong count or flag")
	}

	m.Close(device2, protocol.ErrClosed)

	devs = m.availability("x", "file1", blocks[0].Hash)
	if len(devs) != 0 {
		t.Errorf("Wrong count")
	}

	devs = m.availability("x", "file2", blocks[4].Hash)
	if len(devs) != 1 || devs[device1] != protocol.FlagRequestTemporary {
		t.Errorf("Wrong count or flag")
	}

	devs = m.availability("x", "file3", blocks[6].Hash)
	if len(devs) != 1 || devs[device1] != protocol.FlagRequestTemporary {
		t.Errorf("Wrong count or flag")
	}

	m.Close(device1, protocol.ErrClosed)

	devs = m.availability("x", "file1", blocks[0].Hash)
	if len(devs) != 0 {
		t.Errorf("Wrong count")
	}

	devs = m.availability("x", "file2", blocks[4].Hash)
	if len(devs) != 0 {
		t.Errorf("Wrong count")
	}

	devs = m.availability("x", "file3", blocks[6].Hash)
	if len(devs) != 0 {
		t.Errorf("Wrong count")
	}
}

func TestRequestTempIndex(t *testing.T) {
	db, _ := leveldb.Open(storage.NewMemStorage(), nil)
	m := NewModel(config.Wrap("/tmp/test", config.Configuration{Devices: []config.DeviceConfiguration{{DeviceID: device1}}}), "device", "syncthing", "dev", db)
	m.AddFolder(config.FolderConfiguration{ID: "x", Path: "testdata", Devices: []config.FolderDeviceConfiguration{{DeviceID: device1}}})
	// This we will not share with the device
	m.AddFolder(config.FolderConfiguration{ID: "y", Path: "testdata"})

	fc := &FakeConnection{
		id:           device1,
		indexReturns: protocol.ErrInvalid, // Otherwise it never sends out the temporary index
	}

	state := &sharedPullerState{
		file: protocol.FileInfo{
			Name:   "file",
			Blocks: blocks[1:],
		},
		tempName:  defTempNamer.TempName("file"),
		folder:    "x",
		available: []protocol.BlockInfo{blocks[2], blocks[3], blocks[4], blocks[7]},
	}

	m.progressEmitter.Register(state)

	// This is part of a folder which is unshared with the device
	m.progressEmitter.Register(&sharedPullerState{
		file: protocol.FileInfo{
			Name:   "file",
			Blocks: blocks[1:],
		},
		tempName:  defTempNamer.TempName("file"),
		folder:    "y",
		available: []protocol.BlockInfo{blocks[2], blocks[3], blocks[4], blocks[7]},
	})

	if len(m.progressEmitter.GetTemporaryIndex("x")) != 1 || len(m.progressEmitter.GetTemporaryIndex("y")) != 1 {
		t.Errorf("Incorrect temp index count")
	}

	// This causes us to receive a cluster config message
	m.AddConnection(fc, fc)

	// This causes us to receive indexes, both normal one and temp one.
	m.ClusterConfig(device1, protocol.ClusterConfigMessage{
		ClientName:    "syncthing",
		ClientVersion: "v0.10.20",
		Options: []protocol.Option{
			{
				Key:   "features",
				Value: features(FeatureTemporaryIndex).Marshal(),
			},
		},
	})

	// Give some time for the cluster config message and indexes to arrive to us
	time.Sleep(time.Millisecond)

	if len(fc.clusterConfigs) != 1 || fc.clusterConfigs[0].GetOption("features") != features(FeatureTemporaryIndex).Marshal() {
		t.Errorf("Did not receive cluster config message or did not have right features")
	}

	if len(fc.indexUpdates) != 1 || fc.indexUpdates[0].Flags&protocol.FlagIndexTemporary == 0 {
		t.Errorf("Did not receive temp index or index had a wrong flag")
	}

	blk := fc.indexUpdates[0].Files[0].Blocks[0]

	buf, err := m.Request(device1, "x", "file", blk.Offset, int(blk.Size), blk.Hash, protocol.FlagRequestTemporary, nil)

	hash, verifyErr := scanner.VerifyBuffer(buf, blk)

	if err != nil || verifyErr != nil || !bytes.Equal(hash, blk.Hash) {
		t.Errorf("Wrong response")
	}

	m.progressEmitter.Deregister(state)

	_, err = m.Request(device1, "x", "file", blk.Offset, int(blk.Size), blk.Hash, protocol.FlagRequestTemporary, nil)
	if err == nil {
		t.Errorf("Did not get error as expected")
	}

	// Try get a temp file in an unshared folder
	_, err = m.Request(device1, "y", "file", blk.Offset, int(blk.Size), blk.Hash, protocol.FlagRequestTemporary, nil)
	if err != protocol.ErrNoSuchFile {
		t.Errorf("Did not get error as expected")
	}
}
