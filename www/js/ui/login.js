define([
        "app",
        "jquery",
        "react",
        "lib/minpubsub",
], function(app, $, react, pubsub) {
    var LoginForm = react.createFactory(react.createClass({
                getInitialState: function() {
                    return {name: ""};
                },

                handleSubmit: function(event) {
                    // Avoid default http post
                    event.preventDefault();

                    var name = this.refs.name.getDOMNode().value;
                    var password = this.refs.password.getDOMNode().value;

                    this.refs.name.getDOMNode().value = "";
                    this.refs.password.getDOMNode().value = "";

                    this.props.conn.attemptLogin(name, password);
                },

                handleNameChange: function(event) {
                    this.setState({name: event.target.value});
                },

                render: function() {
                    var disabled = this.props.disabled;

                    var name = this.state.name;

                    var nameId = "name";
                    var passwordId = "password";

                    var loginForm = react.DOM.form({
                            onSubmit: this.handleSubmit,
                            id: "login",
                    },

                    react.DOM.div({},
                        react.DOM.label({
                                htmlFor: nameId,
                        }, "Actor Name"),

                        react.DOM.input({
                                id:  nameId,
                                ref: "name",

                                type:     "text",
                                required: true,
                                value:    name,
                                onChange: this.handleNameChange,
                                disabled: disabled
                        })
                    ),

                    react.DOM.div({},
                        react.DOM.label({
                                htmlFor: passwordId,
                        }, "Password"),

                        react.DOM.input({
                                id:  passwordId,
                                ref: "password",

                                type: "password",
                                required: true,
                                disabled: disabled
                        })
                    ),

                    react.DOM.div({},
                        react.DOM.input({
                                type: "submit",
                                value: "login",
                                disabled: disabled
                        })
                    ));

                    return loginForm;
                }
    }));

    var CreateActorForm = react.createFactory(react.createClass({
                getInitialState: function() {
                    return {password:""};
                },

                handleSubmit: function(event) {
                    // Avoid default http post
                    event.preventDefault();

                    var name = this.props.actor.name;
                    var password = this.props.actor.password;

                    this.props.conn.createActor(name, password);
                },

                handlePasswordChange: function(event) {
                    this.setState({password: event.target.value});
                },

                render: function() {
                    var passwordId = "password";

                    var password = this.state.password;

                    var disabled = true;
                    var color = "red";

                    if (password === this.props.actor.password) {
                        disabled = false;
                        color = "green";
                    }

                    var createActorForm = react.DOM.form({
                            onSubmit: this.handleSubmit,
                            id: "createActor"
                    },
                        react.DOM.div({},
                            react.DOM.span({},
                                "Creating: " + this.props.actor.name
                            )
                        ),

                        react.DOM.div({},
                            react.DOM.label({
                                htmlFor: passwordId
                            }, "Repeat password"),

                            react.DOM.input({
                                    id: passwordId,
                                    ref: "password",

                                    type: "password",
                                    required: true,

                                    onChange: this.handlePasswordChange,

                                    style: { color: color },
                                    value: password
                            })
                        ),

                        react.DOM.div({},
                            react.DOM.input({
                                    type: "submit",
                                    value: "create actor",
                                    disabled: disabled
                            })
                        )
                    );

                    return createActorForm;
                }
    }));

    var LoginUI = function(container) {
        var ui = this;

        ui.on(app.EV_CONNECTED, function(conn) {
            ui.on(app.EV_PACKET, function(packet) {
                console.log(packet);
            });

            ui.on(app.EV_AUTH_FAILED, function(name) {
                console.log("auth failed for", name);
                react.render(new LoginForm({
                            conn:     conn,
                            disabled: false
                }), container).setState({name: name});
            });

            ui.on(app.EV_ACTOR_DOESNT_EXIST, function(name, password) {
                console.log("actor doesn't exist");
                react.render(new CreateActorForm({
                            conn:     conn,
                            actor: {
                                name:     name,
                                password: password
                            },
                            disabled: false
                }), container).setState({password: ""});
            });

            var loginSuccess = function(actor) {
                console.log("login sucess", actor);
            };

            ui.on(app.EV_LOGIN_SUCCESS, loginSuccess);
            ui.on(app.EV_CREATE_SUCCESS, loginSuccess);

            react.render(new LoginForm({conn: conn, disabled: false}), container);
        });
    };

    pubsub(LoginUI.prototype);

    return LoginUI;
});
