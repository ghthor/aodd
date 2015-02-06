define(["client/terrainMap",
       "underscore"
], function(TerrainMap, _) {
    var TerrainMapBuffer = function(def) {
        var map = this;
        map.bounds = def.bounds;

        map.terrain = TerrainMap.parseTerrain(def.terrain);
    };

    var mergeWest = function(slice, map) {
        _.each(map.terrain, function(row, y) {
            row.pop();
            row.unshift(slice.terrain[y][0]);
        });
    };

    var mergeEast = function(slice, map) {
        _.each(map.terrain, function(row, y) {
            row.shift();
            row.push(slice.terrain[y][0]);
        });
    };

    var mergeSouth = function(slice, map) {
        map.terrain.shift();
        map.terrain.push(slice.terrain[0]);
    };

    var mergeNorth  = function(slice, map) {
        map.terrain.pop();
        map.terrain.unshift(slice.terrain[0]);
    };

    var mergeVerticalSlice   = TerrainMap.makeVerticalMerge(mergeWest, mergeEast);
    var mergeHorizontalSlice = TerrainMap.makeHorizontalMerge(mergeSouth, mergeNorth);

    TerrainMapBuffer.prototype = {
        merge: function(slice) {
            return TerrainMap.merge(slice, this, mergeVerticalSlice, mergeHorizontalSlice);
        },

        toDef: function() {
            var terrain = _.map(this.terrain, function(row) {
                return row.join("");
            }).join("\n");

            return {
                bounds: this.bounds,
                terrain: "\n" + terrain + "\n"
            };
        }
    };

    var UpdateBuffer = function() {
        var update = {
            time: 0,
            entities: [],
            removed: []
        };

        this.merge = function(anUpdate) {
            // Error Handling
            if (_.isNull(anUpdate.entities) && _.isNull(anUpdate.removed) && _.isUndefined(anUpdate.terrainMap)) {
                throw "blank update";
            }

            if (update.time >= anUpdate.time) {
                throw "update was in the past";
            }

            update.time = anUpdate.time;

            if (update.entities.length === 0) {
                update.entities = anUpdate.entities;
            } else {
                _.each(anUpdate.entities, function(entity) {
                    update.entities = _.chain(update.entities).reject(function(existing) {
                        return entity.id === existing.id;
                    }).push(entity).value();
                });
            }

            if (update.entities.length === 0) {
                update.removed = update.removed.concat(anUpdate.removed);
            } else {
                _.each(anUpdate.removed, function(removed) {
                    var existingIndex = -1;
                    _.find(update.entities, function(existingEntity, i) {
                        if (existingEntity.id === removed.id) {
                            existingIndex = i;
                            return true;
                        }
                    });

                    if (existingIndex !== -1) {
                        update.entities.splice(existingIndex, 1);
                    } else {
                        update.removed = _.chain(update.removed).reject(function(previouslyRemoved) {
                            return previouslyRemoved.id === removed.id;
                        }).push(removed).value();
                    }
                });
            }

            update.terrainMap = mergeTerrainMap(anUpdate.terrainMap, update.terrainMap);
        };

        var mergeTerrainMap = function(slice) {
            mergeTerrainMap = function(slice, map) {
                if (!_.isUndefined(slice)) {
                    map = map.merge(new TerrainMapBuffer(slice));
                }
                return map;
            };
            // Slice is actually a full map because this is the first message
            return new TerrainMapBuffer(slice);
        };

        this.merged = function() {
            var merged = {
                time:     update.time,
                entities: update.entities,
                removed:  update.removed
            };

            if (!_.isUndefined(update.terrainMap)) {
                merged.terrainMap = update.terrainMap.toDef();
            }
            return merged;
        };
    };

    return UpdateBuffer;
});
