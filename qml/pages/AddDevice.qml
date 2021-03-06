import QtQuick 2.0
import Sailfish.Silica 1.0

Dialog {
    id: addDeviceDialog
    objectName: "addDeviceDialog"

    onDone: {
        if (result == DialogResult.Accepted && !urlField.errorHighlight) {
            if(urlField.text.length > 0) {
                addDevice(urlField.text)
            }
        }
    }

    signal addDevice(string tsurl)

    Column {
        width: parent.width
        spacing: Theme.paddingLarge

        DialogHeader {
            acceptText: "Add"
        }

        Label {
            anchors.horizontalCenter: parent.horizontalCenter
            font.bold: true
            //: Add Device
            //% "Add Device"
            text: qsTrId("whisperfish-add-device")
        }

        TextArea {
            id: urlField
            width: parent.width
            inputMethodHints: Qt.ImhNoPredictiveText
            label: "Device URL"
            placeholderText: "Device URL"
            placeholderColor: Theme.highlightColor
            horizontalAlignment: TextInput.AlignLeft
            EnterKey.onClicked: parent.focus = true
        }

        TextArea {
            anchors.horizontalCenter: parent.horizontalCenter
            width: parent.width
            horizontalAlignment: TextEdit.Center
            readOnly: true
            //: Instructions on how to scan QR code for device linking
            //% "Install Signal Desktop. Use the CodeReader application to scan the QR code displayed on Signal Desktop and copy and paste the URL here."
            text: qsTrId("whisperfish-device-link-instructions")
        }

    }
}
