var app = {};
var db = {};


console.log('doc ready ? ');

$(document).ready(function () {

    console.log('ready');


    $("#origin").change(function () {
        var origin = $(this);
        var URL = origin.val()
        var index = URL.lastIndexOf("/")
        if(index !==-1) {
            var name = URL.substr(index + 1)
            var appName = $('#appName');
            appName.val(name)
        }

    });
    $.ajax({
        dataType: "json",
        url: '/v1/api/meta',
        success: function (resp) {

            if (resp.Status === "ok") {

                resp.sdk.forEach(function (element) {
                    $("#sdk").append(' <option value="' + element + '">' + element + '</option>');
                });

                resp.db.forEach(function (element) {
                    db[element.id] = element;
                    $("#dbEngine").append(' <option value="' + element.id + '">' + element.name + '</option>');
                });

                resp.app.forEach(function (element) {
                    app[element.template] = element;
                    $("#appTemplate").append(' <option value="' + element.template + '">' + element.description + '</option>');
                });


                $("#dbEngine").change(function () {
                    var select = $(this);
                    var dbConfig = $('#dbConfig')
                    var meta = db[select.val()];
                    if (!meta || !meta.hasConfig) {
                        dbConfig.prop("disabled", true);
                        dbConfig.prop("checked", false);
                        return
                    }
                    dbConfig.prop("disabled", false);
                });


                $("#appTemplate").change(function () {
                    var select = $(this);
                    var meta = app[select.val()];
                    var sdk = $('#sdk');
                    var docker = $('#docker');
                    var origin = $('#origin');


                    if (meta.hasOrigin) {
                        origin.val('');
                        origin.prop("disabled", true);

                    } else {
                        origin.prop("disabled", false);

                    }

                    var dbEngine = $('#dbEngine');

                    if (meta.multiDb) {
                        dbEngine.prop("multiple", true);

                    } else {
                        dbEngine.prop("multiple", false);
                    }


                    if (meta.sdk !== "") {
                        sdk.val(meta.sdk);
                        sdk.prop("disabled", true);
                    } else {
                        sdk.prop("disabled", false);
                    }
                    if (meta.docker) {
                        docker.prop("disabled", false);
                    } else {
                        docker.prop("disabled", true);
                        docker.prop("checked", false);
                    }
                })
            }
        }
    });

    $("form").submit(function (e) {
        return submit(e);
    });
});


console.log('doc ready loaded ? ');


function submit(e) {
    var candidates = [
        $('#appTemplate'),
        $('#appName'),
        $('#dbEngine'),
        $('#dbName')
    ];


    var valid = true;
    candidates.forEach(function (element) {
        if (element.required && !isValid(element)) {
            valid = false
        }
    });
    if(! valid) {
        e.preventDefault();
        return false;
    }
    var appName = $('#appName').val();
    appName = appName.replace(/\s/g, '');
    $('form').attr("action", "/download/" + appName + ".zip")
    return true
}


function isValid(element) {
    var hasValue = element.val() !== '';
    if (element.attr('type') === 'checkbox') {
        hasValue = element.is(':checked');
    }

    if (!hasValue) {
        element.removeClass('is-valid');
        element.addClass('is-invalid');
        return false
    }
    element.removeClass('is-invalid');
    element.addClass('is-valid');
    return true
}

