new Vue({
    el: "#app",

    data: {
        ws: null, //our websocket
        newMsg: "", //holds new messages to be sent to the server
        chatContent: "", //A running lost of chat messages displayed on the screen
        username: null, //our usernmae
        avatarUrl: null, //AvatarURL
        joined: false, //true if email and username are filled in
        logs: ""
    },

    created: function () {
        var self = this;
        this.ws = new WebSocket("ws://"+window.location.host+"/ws");
        this.ws.addEventListener("message", function (e) {
            self.logs += 'JSON RECEIVED: '+e.data+'<br/>';
           var msg = JSON.parse(e.data);
           self.chatContent += '<div class="chip">' +
                '<img src="'+self.getAvatarUrl(msg.author.avatarurl)+'">' +//Avatar
                msg.author.username +
               '</div>' +
               emojione.toImage(msg.message)+'<br/>'; //parse emojis

            var element = document.getElementById('chat-messages');
            element.scrollTop = element.scrollHeight; //Auto scroll to the bottom
        });
    },

    methods: {
        send: function () {
            if (this.newMsg !== ""){
                //var self = this;
                //var author = {"username":this.username, "avatarurl": this.avatarUrl};
                //var msg = {"author":author, "message":$('<p>').html(this.newMsg).text()};
                var msgToSend = JSON.stringify({"author":{"username":this.username, "avatarurl": this.avatarUrl}, "message":$('<p>').html(this.newMsg).text()});
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

        join: function () {
            if (!this.username) {
                Materialize.toast('You must choose a username', 2000);
                return
            }
            this.username= $('<p>').html(this.username).text();
            this.avatarUrl = "http://i.imgur.com/tcpgezi.jpg";
            this.joined = true;
        },

        getAvatarUrl: function (url) {
            if (url == null || url === ""){
               return "http://i.imgur.com/tcpgezi.jpg"
            }
            return url;
        }
    }

});