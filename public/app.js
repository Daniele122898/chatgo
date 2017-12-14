/*const(
	MESSAGE_REC = iota 	//0
	ROOM_LIST			//1
	JOINED_ROOM 		//2
	CREATED_ROOM		//3
)*/
const OpEnum ={
    MESSAGE_REC: 0,
    ROOM_LIST: 1,
    JOINED_ROOM: 2,
    CREATED_ROOM: 3,
    LEFT_ROOM: 4
};

var btns = [];

var vue = new Vue({
    el: "#app",

    data: {
        ws: null, //our websocket
        newMsg: "", //holds new messages to be sent to the server
        chatContent: "", //A running lost of chat messages displayed on the screen
        username: null, //our usernmae
        avatarUrl: null, //AvatarURL
        joined: false, //true if email and username are filled in
        joinedRoom: false,
        logs: "",
        rooms: [],
        selectedRoom: null,
        cRoomName: null
    },

    created: function () {
        var self = this;
        this.ws = new WebSocket("ws://"+window.location.host+"/ws");
        this.ws.addEventListener("message", function (e) {
            self.logs += 'JSON RECEIVED: '+e.data+'<br/>';
            var msg = JSON.parse(e.data);
            switch (msg.opcode){
                case OpEnum.MESSAGE_REC:
                    self.msgRec(msg.data);
                    break;
                case OpEnum.ROOM_LIST:
                    self.roomList(msg);
                    break;
                default:
                    Materialize.toast('Got a weird Server Response. View Logs', 2000);
                    break;
            }
        });
    },

    methods: {
        roomList: function (msg) {
            this.rooms = [];
            for (var key in msg["data"]){
                this.rooms.push({"id":key,"name":msg["data"][key].Name})
            }
        },

        msgRec: function (msg) {
            if(msg.author.username === "SYSTEM"){
                this.chatContent += '<div class="systemMsg">'+ msg.message + '</div>'; //Print system Message
            } else {
                this.chatContent += '<div class="chip">' +
                    '<img src="' + this.getAvatarUrl(msg.author.avatarurl) + '">' +//Avatar
                    msg.author.username +
                    '</div>' +
                    emojione.toImage(msg.message) + '<br/>'; //parse emojis
            }

            var element = document.getElementById('chat-messages');
            element.scrollTop =element.scrollHeight; //Auto scroll to the bottom
        },

        send: function () {
            if (this.newMsg !== ""){
                var msgToSend = JSON.stringify({"opcode": OpEnum.MESSAGE_REC,"data": {"author":{"username":this.username, "avatarurl": this.avatarUrl},
                    "message":$('<p>').html(this.newMsg).text(),
                    "roomid":this.selectedRoom.id}});
                this.logs += 'JSON SENT: '+msgToSend+'<br/>';
                this.ws.send(msgToSend);
                this.newMsg = ''; //REset newMsg
            }
        },
        
        changeAv: function () {
            var url = prompt("Please give AvatarURL", "");
            if (url == null || url === "") {
            } else {
                this.avatarUrl = url;
            }
        },

        leaveRoom: function () {
            this.joinedRoom = false;
            this.selectedRoom = null;
            this.chatContent = "";
            var msgToSend = JSON.stringify({"opcode": OpEnum.LEFT_ROOM});
            this.logs += 'LEAVE EVENT JSON: '+msgToSend+'<br/>';
            this.ws.send(msgToSend);
            this.newMsg = "";
            setUpRoomJoiners();
        },

        join: function () {
            if (!this.username) {
                Materialize.toast('You must choose a username', 2000);
                return
            }
            this.username= $('<p>').html(this.username).text();
            this.avatarUrl = "http://i.imgur.com/tcpgezi.jpg";
            this.joined = true;
            var umsg = JSON.stringify({"username":this.username, "avatarurl": ""});
            this.logs += 'JSON REGRISTRATION SENT: '+umsg+'<br/>';
            this.ws.send(umsg);
            setUpRoomJoiners();
        },

        getAvatarUrl: function (url) {
            if (url == null || url === ""){
               return "http://i.imgur.com/tcpgezi.jpg"
            }
            return url;
        },

        createRoom: function () {
            if (!this.cRoomName) {
                Materialize.toast('You must choose a Room name!', 2000);
                return;
            }
            var msgToSend = JSON.stringify({"opcode":OpEnum.CREATED_ROOM, "data" : {"id":"", "name": this.cRoomName}});
            this.logs += 'JSON ROOM CREATION SENT: '+msgToSend+'<br/>';
            this.ws.send(msgToSend);
            this.cRoomName = null;
            setUpRoomJoiners();
        }
    }
});

function setUpRoomJoiners() {
    sleep(1000).then(() => {
        console.log("GETTING BUTTONS");
        btns = document.querySelectorAll(".joinBtn");
        for(var i=0; i<btns.length; i++){
            btns[i].value = i;
            btns[i].addEventListener("click", function () {
                btnSetup(this.value);
            });
        }
    });
}

function sleep (time) {
    return new Promise((resolve) => setTimeout(resolve, time));
}

function btnSetup(num) {
    console.log(num);
    var room = vue.rooms[num];
    var msgToSend = JSON.stringify({"opcode": OpEnum.JOINED_ROOM, "data": {"id": room.id, "name": room.name}});
    vue.selectedRoom = {"id":room.id, "name": room.name};
    vue.logs += 'JSON JOIN EVENT: '+msgToSend+'<br/>';
    vue.ws.send(msgToSend);
    vue.joinedRoom = true;
}