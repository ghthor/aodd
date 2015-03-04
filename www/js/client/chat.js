define(["underscore"], function(_) {
    var Chat = function(socket, pub, entityResolver) {
        var chat = this;

        chat.update = function(update) {
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
    };

    return Chat;
});
