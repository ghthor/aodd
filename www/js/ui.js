// NOTE must be also eddited in specs.js
requirejs.config({
    baseUrl: "js",
    paths: {
        jquery:     "lib/jquery-1.11.2",
        underscore: "lib/underscore",
        react:      "lib/react",
        CAAT:       "lib/caat"
    },
    shim: {
        "underscore": {
            exports: function() {
                return this._.noConflict();
            }
        },
    },
    priority: ["jquery"]
});

require([
        "react",
        "client/client",
        "client/settings",
], function(react, Client, settings) {
    var client = new Client(new WebSocket(settings.websocketURL));

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

                    this.props.client.attemptLogin(name, password);
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

    react.render(new LoginForm({client: client, disabled: true}), document.body);

    client.on("connected", function() {
        console.log("connected to " + settings.websocketURL);

        react.render(new LoginForm({client: client, disabled: false}), document.body);
    });

    client.on("loginSuccess", function(name) {
        console.log(name + " login succeeded");

        react.render(new LoginForm({
                    client: client,
                    disabled: false
        }), document.body).setState({name: name});

        // TODO begin rendering the game here
    });
});
