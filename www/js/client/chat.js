define(["underscore"], function(_) {
    var Chat = function(socket, pub, entityResolver) {
        var chat = this;

        var time = 0;

        chat.update = function(update) {
            time = update.time;

            _.each(update.entities, function(entity) {
                if (!_.isUndefined(entity.type)) {
                    if (entity.type === "say") {
                        pub.emit("chat/say", [
                                entity.id,
                                entityResolver(entity.saidBy).name,
                                entity.msg,
                                entity.saidAt,
                        ]);
                    }
                }
            });
        };

        chat.sendSay= function(msg) {
            socket.send("3::say=" + time + ":" + msg);
        };
    };

    return Chat;
});
