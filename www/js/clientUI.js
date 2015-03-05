define([
        "client/client",
        "react",
        "jquery",
        "underscore",
], function(Client, react, $, _) {
    var takeoverDOM = function(socket, actor) {
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

                        $(this.refs.input.getDOMNode()).blur();
                    },

                    handleChange: function(event) {
                        this.setState({message: event.target.value});
                    },

                    render: function() {
                        return react.DOM.form({
                                onSubmit: this.handleSubmit,
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
                                value: "send",
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
                            chat: this.props.chat,
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

        (function() {
            var client = new Client(socket, actor);

            // Wait for the CAAT director to prepare the canvas
            client.on("ready", function(canvas, inputState, chat) {
                var messages = [];

                var setupKeybinds = function(inputState) {
                    var gameDown = function(e) {
                        var char = String.fromCharCode(e.keyCode);
                        switch (char) {
                        case "W":
                            inputState.movementDown("north");
                            break;
                        case "D":
                            inputState.movementDown("east");
                            break;
                        case "S":
                            inputState.movementDown("south");
                            break;
                        case "A":
                            inputState.movementDown("west");
                            break;
                        default:
                        }

                        switch (e.keyCode) {
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
                            inputState.movementUp("north");
                            break;
                        case "D":
                            inputState.movementUp("east");
                            break;
                        case "S":
                            inputState.movementUp("south");
                            break;
                        case "A":
                            inputState.movementUp("west");
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
                        gameDown(e);
                    });

                    $(document).on("keyup", function(e) {
                        gameUp(e);
                    });
                };

                var render = function() {
                    return react.render(new UI({
                        canvas:        canvas,
                        chat:          chat,
                        messages:      messages,
                    }), document.getElementById("client"));
                };

                // Setup keybinds
                setupKeybinds(inputState);

                client.on("chat/say", function(id, saidBy, msg, saidAt) {
                    messages.push({
                        key:    id,
                        saidBy: saidBy,
                        text:   msg,
                        saidAt: saidAt,
                    });

                    render();
                });

                render();
            });
        }());
    };

    var clientUI = {
        takeoverDOM: takeoverDOM,
    };

    return clientUI;
});
