define(["client/packet",
       "client/imageCache",
       "client/world",
       "client/inputState",
       "client/updateBuffer",
       "lib/minpubsub",
       "jquery",
       "underscore",
       "CAAT"
], function(Packet, ImageCache, World, InputState, UpdateBuffer, pubSub, $, _) {

    // This determines what type of socket is used to send key input
    // browser - WebSocket
    // native  - UDP
    var clientType = _.isUndefined(window.nativeBridge) ? "browser" : "native";

    var Client = function(socket) {
        var client = this;

        this.socket = socket;

        socket.onopen = function() {
            client.onmessage = process.login;
            client.emit("connected");
        };

        socket.onmessage = function(rawPacket) {
            var packet = Packet.Decode(rawPacket.data);
            client.onmessage(packet);
        };

        socket.onerror = function() {
            console.log("Error Connecting to Websocket");
        };

        client.inputState = new InputState(socket);

        var updateBuffer = new UpdateBuffer();

        var process = {
            noop: function(packet) {
                //panic("noop packet processor")
                console.log("noop packet processor", packet);
            },

            login: function(packet) {
                switch(packet.msg) {
                case "loginFailed":
                    client.emit("loginFailed");
                    break;

                case "userDoesntExist":
                    // State change
                    client.onmessage = process.createUser;
                    client.emit("userDoesntExist");
                    break;

                case "loginSuccess":
                    // State change
                    client.onmessage = process.selectChar;
                    client.emit("loginSuccess", [packet.payload]);
                    client.emit("selectChar", [packet.payload]);
                    break;

                default:
                    console.log("Unexpected Packet during `login`", packet);
                }
            },

            createUser: function(packet) {
                switch(packet.msg) {
                case "mustProvidePasswd":
                    //panic("recieved mustProvidePasswd")
                    break;

                case "userAlreadyExists":
                    //panic("recieved userAlreadyExists")
                    break;

                case "userCreated":
                    // State change
                    client.onmessage = process.createChar;
                    client.emit("userCreated");
                    client.emit("createChar");
                    break;

                default:
                    console.log("Unexpected Packet during `createUser`", packet);
                }
            },

            createChar: function(packet) {
                switch(packet.msg) {
                case "charAlreadyExists":
                    // TODO
                    break;

                case "charCreated":
                    // State change
                    client.onmessage = process.bufferUpdates;
                    client.playerEntity = JSON.parse(packet.payload);
                    client.emit("charCreated");
                    client.emit("loadGame");
                    break;

                default:
                    console.log("Unexpected Packet during `createChar`", packet);
                }
            },

            selectChar: function(packet) {
                switch(packet.msg) {
                case "charDoesntExist":
                    //panic("recieved charDoesntExist")
                    break;

                case "charSelected":
                    // State change
                    client.onmessage = process.bufferUpdates;
                    client.playerEntity = JSON.parse(packet.payload);
                    client.emit("charSelected");
                    client.emit("loadGame");
                    break;

                default:
                    console.log("Unexpected Packet during `selectChar`", packet);
                }
            },

            bufferUpdates: function(packet) {
                if (packet.msg === "update") {
                    updateBuffer.merge(JSON.parse(packet.payload));
                } else {
                    console.log(packet);
                }
            },

            update: function(packet) {
                if (packet.msg === "update") {
                    var worldState = JSON.parse(packet.payload);
                    client.world.update(worldState);
                    client.inputState.update(worldState.time);
                } else {
                    console.log(packet);
                }
            }
        };

        client.onmessage = process.noop;

        var imageCache = new ImageCache();

        var imagesFinishedLoading = function(imageCache) {
            CAAT.DEBUG = 1;

            var $stage = $("#stage");

            // Create the Director and Scene
            var director = new CAAT.Director().
                initialize($stage.innerWidth(), $stage.innerHeight()).
                setImagesCache(imageCache);
            var scene = director.createScene().setFillStyle("#c0c0c0");

            client.director = director;
            client.scene = scene;

            if (!_.isUndefined(imagesFinishedLoading.after)) {
                imagesFinishedLoading.after(director, scene);
            }
        };

        imageCache.on("complete", imagesFinishedLoading);

        client.beginRendering = function() {
            var startRendering = function(director, scene) {

                client.world = new World(director, scene, client.playerEntity);
                client.world.update(updateBuffer.merged());
                client.onmessage = process.update;

                // Setup keybinds
                $(document).on("keydown", function(e) {
                    var char = String.fromCharCode(e.keyCode);
                    switch (char) {
                        case "W":
                            client.inputState.movementDown("north");
                        break;
                        case "D":
                            client.inputState.movementDown("east");
                        break;
                        case "S":
                            client.inputState.movementDown("south");
                        break;
                        case "A":
                            client.inputState.movementDown("west");
                        break;
                        default:
                    }

                }).on("keyup", function(e) {
                    var char = String.fromCharCode(e.keyCode);
                    switch (char) {
                        case "W":
                            client.inputState.movementUp("north");
                        break;
                        case "D":
                            client.inputState.movementUp("east");
                        break;
                        case "S":
                            client.inputState.movementUp("south");
                        break;
                        case "A":
                            client.inputState.movementUp("west");
                        break;
                        default:
                    }
                });

                client.emit("canvasReady", [director.canvas]);
                CAAT.loop();
            };

            // TODO Experiment loading images at different points
            // * Before User Logged in - Fastest, but uses more bandwidth if the user leaves before playing
            // * During Character Select - Probly best choice in a balance between speed and bandwidth waste
            // * Right here - Slowest load since we'll have many WorldStates before we're ready to render
            imageCache.loadDefault();

            if (_.isUndefined(client.director)) {
                // Images haven't finished loading yet
                imagesFinishedLoading.after = startRendering;
            } else {
                // Images Finished loading, we're ready to go
                startRendering(client.director, client.scene);
            }
        };

        return this;
    };

    Client.prototype = {
        clientType: clientType,
        attemptLogin: function(name, passwd) {
            var packet = Packet.JSON("login", {
                name: name,
                passwd: passwd
            });
            this.socket.send(Packet.Encode(packet));
        },
        
        createUser: function(name, passwd) {
            var packet = Packet.JSON("createUser", {
                name: name,
                passwd: passwd
            });
            this.socket.send(Packet.Encode(packet));
        },

        selectChar: function(name) {
            var packet = {
                type: Packet.Type.PT_MESSAGE,
                msg: "selectChar",
                payload: name
            };
            this.socket.send(Packet.Encode(packet));
        },

        createChar: function(name) {
            var packet = {
                type: Packet.Type.PT_MESSAGE,
                msg: "createChar",
                payload: name
            };
            this.socket.send(Packet.Encode(packet));
        }
    };

    // Extend Client to be a message publisher generator
    pubSub(Client.prototype);

    return Client;
});
