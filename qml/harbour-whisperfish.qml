import QtQuick 2.0
import Sailfish.Silica 1.0
import org.nemomobile.contacts 1.0
import org.nemomobile.notifications 1.0
import "pages"

ApplicationWindow
{
    id: mainWindow
    cover: Qt.resolvedUrl("cover/CoverPage.qml")
    initialPage: Component { Main { } }
    allowedOrientations: Orientation.All
    _defaultPageOrientations: Orientation.All
    _defaultLabelFormat: Text.PlainText

    ImagePicker {
        id: imagepicker
    }

    PeopleModel {
        id: peopleModel
        filterType: PeopleModel.FilterNone
    }

    Component {
        id: messageNotification
        Notification {}
    }

    function activateSession(sid, name, source) {
        console.log("Activating session for source: "+source)
        SessionModel.markRead(sid)
        MessageModel.load(sid, name)
    }

    function newMessageNotification(sid, name, source, message, isGroup) {
        if(Qt.application.state == Qt.ApplicationActive &&
           (pageStack.currentPage.objectName == "main" ||
           (sid == MessageModel.sessionId && pageStack.currentPage.objectName == "conversation"))) {
           return
        }

        var m = messageNotification.createObject(null)
        if(SettingsBridge.boolValue("show_notify_message")) {
            m.body = message
        } else {
            //: Default label for new message notification
            //% "New Message"
            m.body = qsTrId("whisperfish-notification-default-message")
        }
        m.category = "harbour-whisperfish-message"
        m.previewSummary = name
        m.previewBody = m.body
        m.summary = name
        m.clicked.connect(function() {
            console.log("Activating session: "+sid)
            mainWindow.activate()
            showMainPage()
            mainWindow.activateSession(sid, name, source)
            pageStack.push(Qt.resolvedUrl("pages/Conversation.qml"), {}, PageStackAction.Immediate)
        })
        // This is needed to call default action
        m.remoteActions = [ {
            "name": "default",
            "displayName": "Show Conversation",
            "icon": "harbour-whisperfish",
            "service": "org.whisperfish.session",
            "path": "/message",
            "iface": "org.whisperfish.session",
            "method": "showConversation",
            "arguments": [ "sid", sid ]
        } ]
        m.publish()
    }

    Connections {
        target: ClientWorker
        onMessageReceived: {
            if(sid == MessageModel.sessionId && pageStack.currentPage.objectName == "conversation") {
                SessionModel.add(sid, true)
                MessageModel.add(mid)
            } else {
                SessionModel.add(sid, false)
            }
        }
        onMessageReceipt: {
            if(mid > 0 && pageStack.currentPage.objectName == "conversation") {
                MessageModel.markReceived(mid)
            }

            if(sid > 0) {
                SessionModel.markReceived(sid)
            }
        }
        onNotifyMessage: {
            newMessageNotification(sid, ContactModel.name(source), source, message, isGroup)
        }
    }

    Connections {
        target: SendWorker
        onMessageSent: {
            if(sid == MessageModel.sessionId && pageStack.currentPage.objectName == "conversation") {
                SessionModel.markSent(sid, message)
                MessageModel.markSent(mid)
            } else {
                SessionModel.markSent(sid, message)
            }
        }
        onPromptResetPeerIdentity: {
            pageStack.push(Qt.resolvedUrl("pages/ResetPeerIdentity.qml"), { source: source })
        }
    }

    function showMainPage() {
        pageStack.clear()
        pageStack.push(Qt.resolvedUrl("pages/Main.qml"), {}, PageStackAction.Immediate)
    }

    function newMessage(operationType) {
        showMainPage()
        pageStack.push(Qt.resolvedUrl("pages/NewMessage.qml"), { }, operationType)
    }

    function logSectionHeaders() {
        //: Session section label for today
        //% "Today"
        console.log(qsTrId("whisperfish-session-section-today"))

        //: Session section label for yesterday
        //% "Yesterday"
        console.log(qsTrId("whisperfish-session-section-yesterday"))

        //: Session section label for older
        //% "Older"
        console.log(qsTrId("whisperfish-session-section-older"))
    }
}
