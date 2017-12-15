

function validateEmail(email) {
    var re = /^(([^<>()[\]\\.,;:\s@\"]+(\.[^<>()[\]\\.,;:\s@\"]+)*)|(\".+\"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$/;
    return re.test(email);
}

function setSystemError(error) {
    console.log(error)
    var errorElement = $('#errorMessage');
    errorElement.text('Ups, something went wrong. Try again later.')
    errorElement.addClass('text-danger')
}


function clearErrors() {
    var email = $('#email');
    var name = $('#name');
    var password = $('#password');
    var retypedPassword = $('#retypedPassword');
    var dateOfBirth = $('#dateOfBirth');
    clearElementError(email);
    clearElementError(name);
    clearElementError(password);
    clearElementError(retypedPassword);
    clearElementError(dateOfBirth);
    $('#errorMessage').text('');
}


function signup() {

    var email = $('#email');
    var name = $('#name');
    var password = $('#password');
    var retypedPassword = $('#retypedPassword');
    var dateOfBirth = $('#dateOfBirth');
    var data = {
        email:email.val(),
        name:name.val(),
        password:password.val(),
        dateOfBirth:dateOfBirth.val(),
        landingPage:'/signin/'
    };

    var hasError = false;
    if(checkIfEmpty(email)) {
        if(!validateEmail(email.val())) {
            setElementError(email, "$label does not look like a valid email")
            hasError = true;
        } else {
            clearElementError(email);
        }
    } else {
        hasError = true;
    }

    if(!checkIfEmpty(name)) {
        hasError = true;
    }

    if(!checkIfEmpty(password)) {
        hasError = true;
    } else {
        if(retypedPassword.val() !== password.val()) {
            setError(retypedPassword, "Password does not match")
        } else {
            clearElementError(retypedPassword);
        }
    }
    if(!checkIfEmpty(dateOfBirth)) {
        hasError = true;
    }
    if(hasError) {
        return;
    }

    $.ajax({
        type: "POST",
        url: "/api/singup/",
        data: JSON.stringify(data),
        contentType: "application/json; charset=utf-8",
        dataType: "json",
        success: function (data) {

            if(data.status === 'ok') {
                if(data.landingPage) {
                    window.location.href = data.landingPage;
                }
            } else {

                clearErrors();
                var errorSource = data.errorSource;
                if(errorSource === 'system') {
                    setSystemError(data.error)
                } else {
                    var element = $('#' + errorSource)
                    setElementError(element, data.error)
                }
            }

            console.log('success');
            console.log(data);


        },
        error: function(response) {
            if(response.status != 200) {
                setSystemError(response.responseText)
            }
        }
    });


}


$(document).ready(function () {
    $("form").submit(function (e) {
        e.preventDefault();
        signup();
    });
    // $('#email').val('abc@ew.pl');
    // $('#name').val('asds');
    // $('#password').val('abc');
    // $('#retypedPassword').val('abc');
    // $('#dateOfBirth').val('1987-04-02')

});




function setElementError(element, message) {
    element.addClass('form-control-danger');
    element.parent().addClass('has-danger');
    var small = element.parent().find('small');
    if(message.indexOf("$label") !=-1) {
        var label = element.parent().parent().find('label').text()
        message = message.replace("$label", label)
    }
    small.text(message)
}

function clearElementError(element, message) {
    element.removeClass('form-control-danger');
    element.parent().removeClass('has-danger')
    element.removeClass('form-control-success');
    element.parent().addClass('has-success');
    small = element.parent().find('small');
    small.text();
}


