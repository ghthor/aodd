define([
        "ui/canvas",

        // TODO port these into gopherjs
        "ui/client/input_state",
        "ui/client/chat",

        "app",
        "github.com/ghthor/engine/rpg2d/coord",

        "react",
        "jquery",
        "underscore",
        "lib/minpubsub",
], function(Canvas, InputState, Chat, app, coord, react, $, _, pubsub) {
        var Message = react.createFactory(react.createClass({
                    render: function() {
                        return react.DOM.li({
                                className: "chat-message",
                        }, react.DOM.span({
                                className: "chat-message-said-by",
                        }, this.props.saidBy),
                        " says, \"",
                        react.DOM.span({
                                className: "chat-message-text",
                        }, this.props.text),
                        "\"");
                    },
        }));

        var MessageList = react.createFactory(react.createClass({
                    componentDidUpdate: function() {
                        var list = this.refs.messageList.getDOMNode();

                        if (list.scrollHeight - (list.scrollTop + list.offsetHeight) < 50) {
                            list.scrollTop = list.scrollHeight;
                        }
                    },

                    render: function() {
                        var messages = _.map(this.props.messages, function(m) {
                            return new Message(m);
                        });

                        return react.DOM.ul({
                                ref: "messageList",
                                className: "chat-message-list",
                        }, messages);
                    },
        }));

        var Input = react.createFactory(react.createClass({
                    getInitialState: function() {
                        return {message: ""};
                    },

                    handleSubmit: function(event) {
                        event.preventDefault();

                        var msg = this.refs.input.getDOMNode().value;
                        if (msg === "") {
                            return;
                        }

                        this.refs.input.getDOMNode().value = "";
                        this.setState({message: ""});

                        this.props.chat.sendSay(msg);
                    },

                    handleChange: function(event) {
                        this.setState({message: event.target.value});
                    },

                    componentDidUpdate: function() {
                        if (this.props.display) {
                            $(this.refs.input.getDOMNode()).focus();
                        } else {
                            $(this.refs.input.getDOMNode()).blur();
                        }
                    },

                    render: function() {
                        var style = {
                            visibility: "hidden",
                        };

                        if (this.props.display) {
                            delete style.visibility;
                        }

                        return react.DOM.form({
                                onSubmit: this.handleSubmit,
                                style: style,
                        },

                        react.DOM.input({
                                ref: "input",
                                className: "chat-input",

                                type: "text",
                                value: this.state.message,

                                onChange: this.handleChange,
                        }),

                        react.DOM.input({
                                type: "submit",
                                value: "say",
                        }));
                    },
        }));

        var ChatBox = react.createFactory(react.createClass({
                render: function() {
                    var messages = this.props.messages;

                    return react.DOM.div({
                            className: "chat-box",
                            ref: "root",
                    }, new MessageList({
                            messages: messages,
                    }), new Input({
                            chat:        this.props.chat,
                            display:     this.props.chatDisplayed,
                    }));
                },
        }));

        var UI = react.createFactory(react.createClass({
            componentDidMount: function() {
                var div = this.getDOMNode();
                $(div).prepend(this.props.canvas);
            },

            render: function() {
                return react.DOM.div({}, new ChatBox(this.props));
            },
        }));

    var Client = function(container, loggedInConn) {
        var client = this;

        // Wait for the CAAT director to prepare the canvas
        var canvasReady = function(canvas) {
            client.on(app.EV_RECV_INPUT_CONN, function(inputConn) {
                var messages = [];
                var chatDisplayed = false;

                // Create a new input state manager
                var inputState = new InputState(inputConn);

                var chat = (function() {
                    var eventPublisher = client;
                    return new Chat(inputConn, eventPublisher, function(entityId) {
                        return entityId;
                    });
                }());

                var render = function() {
                    return react.render(new UI({
                        canvas:        canvas,
                        chat:          chat,
                        chatDisplayed: chatDisplayed,
                        messages:      messages,
                    }), container);
                };

                var setupKeybinds = function(inputState) {
                    var gameDown = function(e) {
                        var char = String.fromCharCode(e.keyCode);
                        switch (char) {
                        case "W":
                            inputState.movementDown(coord.North);
                            break;
                        case "D":
                            inputState.movementDown(coord.East);
                            break;
                        case "S":
                            inputState.movementDown(coord.South);
                            break;
                        case "A":
                            inputState.movementDown(coord.West);
                            break;
                        default:
                        }

                        switch (e.keyCode) {
                        case 13: // enter in chromium
                            chatDisplayed = true;
                            render();
                            break;
                        case 32: // space in chromium
                            inputState.assailDown();
                            break;
                        default:
                        }

                    };

                    var gameUp = function(e) {
                        var char = String.fromCharCode(e.keyCode);
                        switch (char) {
                        case "W":
                            inputState.movementUp(coord.North);
                            break;
                        case "D":
                            inputState.movementUp(coord.East);
                            break;
                        case "S":
                            inputState.movementUp(coord.South);
                            break;
                        case "A":
                            inputState.movementUp(coord.West);
                            break;
                        }

                        switch (e.keyCode) {
                        case 32: // space in chromium
                            inputState.assailUp();
                            break;
                        default:
                        }
                    };

                    $(document).on("keydown", function(e) {
                        if (!chatDisplayed) {
                            gameDown(e);
                        }
                    });

                    $(document).on("keyup", function(e) {
                        if (!chatDisplayed) {
                            gameUp(e);
                        }
                    });
                };

                // Setup keybinds
                setupKeybinds(inputState);

                // TODO These listeners should be triggered by gopherjs
                client.on("chat/recv/say", function(id, saidBy, msg, saidAt) {
                    messages.push({
                        key:    id,
                        saidBy: saidBy,
                        text:   msg,
                        saidAt: saidAt,
                    });

                    render();
                });

                // TODO These listeners should be triggered by gopherjs
                client.on("chat/sent/say", function() {
                    chatDisplayed = false;
                    render();
                });

                render();
            });

            loggedInConn.connectActor(client);
        };

        client.render = function() {
            var canvas = new Canvas(client);
            canvas.on("ready", canvasReady);
        };

        return this;
    };

    pubsub(Client.prototype);

    return Client;
});
