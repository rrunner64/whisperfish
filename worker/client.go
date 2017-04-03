// Copyright 2016 Andrew E. Bruno
//
// This file is part of Whisperfish.
//
// Whisperfish is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Whisperfish is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Whisperfish.  If not, see <http://www.gnu.org/licenses/>.

package worker

import (
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/aebruno/textsecure"
	"github.com/aebruno/whisperfish/settings"
	"github.com/aebruno/whisperfish/store"
	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/network"
)

//go:generate qtmoc
type ClientWorker struct {
	core.QObject

	settings *settings.Settings
	manager  *network.QNetworkConfigurationManager

	_ bool                                                  `property:"connected"`
	_ func()                                                `constructor:"init"`
	_ func()                                                `signal:"disconnected"`
	_ func()                                                `signal:"reconnect"`
	_ func()                                                `signal:"startConnection"`
	_ func(sid int64, mid int64)                            `signal:"messageReceived"`
	_ func(sid int64, mid int64)                            `signal:"messageReceipt"`
	_ func(sid int64, source, message string, isGroup bool) `signal:"notifyMessage"`
}

func (c *ClientWorker) init() {
	c.settings = settings.NewSettings(nil)
	c.manager = network.NewQNetworkConfigurationManager(nil)
	c.SetConnected(false)
	c.ConnectReconnect(c.reconnect)
	c.ConnectStartConnection(func() {
		c.manager.ConnectConfigurationChanged(func(config *network.QNetworkConfiguration) {
			// If we change network configurations (i.e. WLAN to Cellular) force reconnect
			if config.State() == network.QNetworkConfiguration__Active {
				c.reconnect()
			}
		})

		go c.start()
	})
}

// Reconnect to Signal
func (c *ClientWorker) reconnect() {
	log.Info("Forcing reconnect of websockets")
	textsecure.StopListening()
}

// Start websocket listener
func (c *ClientWorker) start() {
	for {
		time.Sleep(3 * time.Second)

		if !c.manager.IsOnline() {
			log.Debug("No network connection found")
			continue
		}

		log.Debug("Starting client websocket listener")
		c.SetConnected(true)

		if err := textsecure.StartListening(); err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("Error processing Websocket event from Signal")
		}

		c.SetConnected(false)
	}
}

// Process incoming message from Signal
func (c *ClientWorker) MessageHandler(msg *textsecure.Message, isSyncSent bool, ts uint64) {
	log.WithFields(log.Fields{
		"source":     msg.Source(),
		"isSyncSent": isSyncSent,
		"ts":         ts,
	}).Info("Message received")

	message := &store.Message{
		Source:  msg.Source(),
		Message: msg.Message(),
		Flags:   msg.Flags(),
	}

	if isSyncSent {
		message.Outgoing = true
		message.Sent = true
		if ts > 0 {
			message.Timestamp = ts
		}
	} else {
		message.Timestamp = msg.Timestamp()
	}

	if len(msg.Attachments()) > 0 {
		if c.settings.GetBool("save_attachments") && !c.settings.GetBool("incognito") {
			err := message.SaveAttachment(c.settings.GetString("attachment_dir"), msg.Attachments()[0])
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
				}).Error("Failed to save attachment")
			}
		} else {
			message.HasAttachment = true
			message.MimeType = msg.Attachments()[0].MimeType
		}
	}

	if msg.Group() != nil && msg.Group().Flags == textsecure.GroupUpdateFlag {
		message.Message = "Member joined group"
	} else if msg.Group() != nil && msg.Group().Flags == textsecure.GroupLeaveFlag {
		message.Message = "Member left group"
	}

	sess, err := store.DS.ProcessMessage(message, msg.Group(), !message.Sent)
	if err != nil {
		log.WithFields(log.Fields{
			"err":        err,
			"source":     msg.Source(),
			"isSyncSent": isSyncSent,
			"ts":         ts,
		}).Info("Failed to process incoming message")
	}

	c.MessageReceived(sess.ID, message.ID)
	c.NotifyMessage(sess.ID, sess.Source, sess.Message, sess.IsGroup)
}

// Receipt handler
func (c *ClientWorker) ReceiptHandler(source string, devID uint32, ts uint64) {
	log.WithFields(log.Fields{
		"source":    source,
		"timestamp": ts,
		"devID":     devID,
	}).Debug("Receipt handler")

	var err error
	sessionID := int64(0)
	messageID := int64(0)
	tries := 0

	for {
		sessionID, messageID, err = store.DS.MarkMessageReceived(source, ts)
		if err != nil {
			tries++
			if tries > 3 {
				log.WithFields(log.Fields{
					"error":     err,
					"source":    source,
					"timestamp": ts,
				}).Error("Failed to mark message received")
				return
			}
			log.Debug("receiptHandler can't find message. Trying again later")
			time.Sleep(500 * time.Millisecond)
			continue
		}
		break
	}

	c.MessageReceipt(sessionID, messageID)
}

/*
    XXX This is currently a bug in the go qt bindings. Including this comment
    is dirty fix for unimplemented pure virtual functions. This will be removed
    once a better parser is implemented. For more information see here:
    https://github.com/therecipe/qt/issues/220

   .Duration(
   .UpdateCurrentTime(
*/
