$(document).ready(function () {

    $.ajax({
        dataType: "json",
        url: '/v1/api/dummytype',
        success: function (resp) {

            console.log(resp)
            for (var i = 0; i < resp.Data.length; i++) {
                var item = resp.Data[i];
                $("#typeId").append(' <option value="' + item.Id + '">' + item.Name + '</option>');
            }
        }
    });

    $("#submit").click(function (e) {
        e.preventDefault();
        submit();
    });
});


function submit() {
    var id = $('#id');
    var name = $('#name');
    var typeId = $('#typeId');
    var agree = $('#agree');

    var valid = true;
    $('.form-control').each(function (index, element) {
        console.log(element);
        if(element.required && ! isValid($(this))) {
            valid = false
        }
    });
    if(!valid) {
        return false
    }

    var request = {
        Data: {
            Name: name.val(),
            TypeId: parseInt(typeId.val())
        }
    };
    if (id.val() !== '') {
        request.Data.Id = parseInt(id.val())
    }

    $.ajax({
        type: "POST",
        url: "/v1/api/dummy",
        data: JSON.stringify(request),
        contentType: "application/json; charset=utf-8",
        dataType: "json",
        success: function (data) {
            if (data.Status === 'ok') {
                window.location.href = '/dummy.html'
            } else {
                clearErrors();
                setSystemError(data.error)
            }
        },
        error: function (response) {
            if (response.status !== 200) {
                setSystemError(response.responseText)
            }
        }
    });
    return true
}

function isValid(element) {
    var hasValue =  element.val() !== '';


    if(element.attr('type')=== 'checkbox') {
        hasValue = element.is(':checked');
    }


    if (! hasValue) {
        element.removeClass('is-valid');
        element.addClass('is-invalid');
        return false
    }
    element.removeClass('is-invalid');
    element.addClass('is-valid');
    return true
}



function setSystemError(error) {
    console.log(error)
    var errorElement = $('#errorMessage');
    errorElement.text('Ups, something went wrong. Try again later.')
    errorElement.addClass('text-danger')
}