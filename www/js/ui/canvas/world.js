// TODO This module is 100% unspecified
define(["underscore",
       "ui/canvas/sprite/human",
       "ui/canvas/bar",
       "ui/canvas/chat_bubble",
       "ui/canvas/player",
       "CAAT",
], function(_, Human, Bar, Bubble, Player) {
    var World = function(director, scene) {
        var world = this;

        world.grid = 16;
        var grid = world.grid,
            // TODO should be set from "client/settings" module
            width  = 2001 * grid,
            height = 2001 * grid;

        world.scene = scene;

        // This container includes all the CAAT.Actor's
        // for all the entities in the world
        var container = new CAAT.ActorContainer().
            setSize(width, height).
            setPositionAnchored(scene.width/2, scene.height/2, 0.5, 0.5);

        scene.addChild(container);

        world.destroy = function() {
            container.destroy();
        };

        var cellToLocal = world.cellToLocal = function(cell) {
            return {
                x:  width/2 + cell.X * grid,
                y: height/2 - cell.Y * grid
            };
        };

        var newWall = function(entity) {
            var p = cellToLocal(entity.Cell);
            var actor = new CAAT.ActorContainer().
                setSize(grid, grid).
                setPositionAnchored(p.x, p.y, 0.5, 0.5);

            var img = new CAAT.SpriteImage().initialize(director.getImage("rock"), 1, 1);

            var sprite = new CAAT.Actor().
                setBackgroundImage(img.getRef(), true).
                setPositionAnchored(actor.width/2, actor.height/2, 0.5, 0.5);

            actor.addChild(sprite);
            return actor;
        };

        var newActor = function(entity) {
            var p = cellToLocal(entity.Cell);
            var actor = new CAAT.ActorContainer().
                setSize(grid, grid).
                setPositionAnchored(p.x, p.y, 0.5, 0.5);


            var animation = new Human(director).
                makeSprites(Human.makeImage(director, "female"), actor.width/2, actor.height/2);
            _.each(animation.sprites, function(sprite) {
                actor.addChild(sprite);
            });

            var name = new CAAT.TextActor().
                setBounds(0, -grid, grid, grid).
                setTextAlign("center").
                setText(entity.Name);
            actor.addChild(name);

            var healthBar = new Bar(grid, grid/4, "red");
            healthBar.actor.
                setPositionAnchored(grid/2, -grid/2+4, 0.5, 0.5);
            actor.addChild(healthBar.actor);

            actor.setHealthPercentage = function(percent) {
                healthBar.setPercent(percent);
            };

            var bubble = new Bubble(150, 80);
            bubble.actor.setPositionAnchored(grid/2, -10, 0.5, 1);
            actor.addChild(bubble.actor);

            actor.setSayMsg = function(id, msg) {
                bubble.setMsg(id, msg);
            };
            actor.clearSayMsg = function(id) {
                bubble.clearMsg(id);
            };

            actor.setAnimation = function(entity) { animation.setAnimation(entity); };
            return actor;
        };

        var actorSetMovement = function(actor, orig, dest, duration) {
            orig = cellToLocal(orig);
            dest = cellToLocal(dest);
            var path = new CAAT.LinearPath().
                setInitialPosition(orig.x, orig.y).
                setFinalPosition(dest.x, dest.y);

            var behavior = new CAAT.PathBehavior().
                setPath(path).
                setDelayTime(0, duration * 1000/40);

            actor.emptyBehaviorList();
            actor.addBehavior(behavior);
        };

        var actorUpdatePosition = function(actor, entity) {
            var p = cellToLocal(entity.Cell);
            actor.emptyBehaviorList();
            actor.setPositionAnchored(p.x, p.y, 0.5, 0.5);
        };

        var playerSetPathAction = function(orig, dest, duration) {
            orig = {
               x: -orig.X * grid + scene.width/2,
               y:  orig.Y * grid + scene.height/2
            };

            dest = {
               x: -dest.X * grid + scene.width/2,
               y:  dest.Y * grid + scene.height/2
            };

            // TODO remove the 40fps hardcoded server fps
            container.emptyBehaviorList().
                addBehavior(new CAAT.PathBehavior().
                    setPath(new CAAT.LinearPath().
                        setInitialPosition(orig.x, orig.y).
                        setFinalPosition(dest.x, dest.y)).
                    setDelayTime(0, duration * 1000.0/40.0));
        };

        var playerSetPosition = function(cell) {
            // Convert the cell to a canvas point
            var p = {
                x: -cell.X * grid + scene.width/2,
                y:  cell.Y * grid + scene.height/2,
            };

            container.emptyBehaviorList().
                setPosition(p.x, p.y);
        };

        world.initialize = function(playerEntity, worldState, terrain) {
            var time = worldState.time;

            // Index of all entities currently being displayed
            var entities = {};
            var actors = {};

            var player = new Player({
                    director:   director,
                    scene:      scene,
                    gridSz:     grid,

                    setPathAction: playerSetPathAction,
                    setPosition:   playerSetPosition,
            });

            player.initialize(time, playerEntity);

            var removeEntity = function(entity) {
                entity = entities[entity.Id];

                if (_.isUndefined(entity)) {
                  return; // continue
                }

                if (!_.isUndefined(entity.Type)) {
                    if (entity.Type === "say") {
                        if (player.is(entity.SaidBy)) {
                            player.clearSayMsg(entity.Id);
                        } else {
                            var actor = actors[entity.SaidBy];
                            if (!_.isUndefined(actor)) {
                                actor.clearSayMsg(entity.Id);
                            }
                        }
                    }

                    delete entities[entity.Id];
                    return; //continue
                }

                var actor = actors[entity.Id];
                actor.destroy();

                delete entities[entity.Id];
                delete actors[entity.Id];

                console.log(entity);
            };

            var updateEntity = function(entity) {
                if (player.is(entity.Id)) {
                    player.update(time, entity);
                    return; //continue
                }

                if (!_.isUndefined(entity.Type)) {
                    if (entity.Type === "assail") {
                        (new Audio("asset/audio/assail.wav")).play();
                    }

                    if (entity.Type === "say") {
                        if (player.is(entity.SaidBy)) {
                            player.setSayMsg(entity.Id, entity.Msg);
                        } else {
                            var actor = actors[entity.SaidBy];
                            if (!_.isUndefined(actor)) {
                                actor.setSayMsg(entity.Id, entity.Msg);
                            }
                        }
                        entities[entity.Id] = entity;
                    }

                    if (entity.Type === "wall") {
                        (function() {
                            var actor = newWall(entity);
                            container.addChild(actor);
                            entities[entity.Id] = entity;
                            actors[entity.Id] = actor;
                        }());
                    }

                    if (entity.Type === "removed") {
                        removeEntity(entity);
                    }

                    return; //continue
                }

                // Get a handle to the CAAT.Actor
                var actor;
                if (_.isUndefined(entities[entity.Id])) {
                    // Create a new CAAT.Actor
                    actor = newActor(entity);
                    container.addChild(actor);
                    actors[entity.Id] = actor;
                } else {
                    actor = actors[entity.Id];
                }

                entities[entity.Id] = entity;

                // Check if the entity is moving
                if (!_.isNull(entity.PathAction)) {
                    var pa = entity.PathAction;
                    // TODO Fix display of an entity that has
                    //      a pathAction.start < time but
                    //      this is the first time we've seen it
                    //      so it doesn't have a CAAT.Action
                    //      to it.
                    if (pa.Start === time) {
                        var duration = pa.End - pa.Start;
                        actorSetMovement(actor, pa.Orig, pa.Dest, duration);
                    }

                } else {
                    actorUpdatePosition(actor, entity);
                }
                
                // Update animation
                actor.setAnimation(entity);

                // update health display
                actor.setHealthPercentage(entity.Hp/entity.HpMax);
            };

            // Update all entities
            _.each(worldState.EntitiesNew, updateEntity);
            _.each(worldState.EntitiesChanged, updateEntity);
            _.each(worldState.EntitiesUnchanged, updateEntity);

            // Create an actor for the rendered terrain map
            var renderedTerrain = new CAAT.Actor().setBackgroundImage(terrain.canvas, true);
            container.addChildAt(renderedTerrain, 0);

            var terrainSetPosition = function(cell) {
                // Moves like an entity in the world
                var p = cellToLocal(cell);
                renderedTerrain.setPositionAnchored(p.x, p.y, 0.5, 0.5);
            };
            terrainSetPosition(terrain.map.center());

            world.update = function(worldStateDiff) {
                time = worldStateDiff.Time;

                // Update all entities
                _.each(worldStateDiff.Entities, updateEntity);
                // Remove entities that don't exist anymore
                _.each(worldStateDiff.Removed, removeEntity);

                if (!_.isNull(worldStateDiff.TerrainMapSlices)) {
                    if (worldStateDiff.TerrainMapSlices.length > 0) {
                        terrainSetPosition(terrain.map.center());
                    }
                }
            };
        };

        return this;
    };

    return World;
});
