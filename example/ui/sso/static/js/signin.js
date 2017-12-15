


function setSystemError(error) {
    console.log(error);
    var errorElement = $('#errorMessage');
    errorElement.text('Ups, something went wrong. Try again later.')
    errorElement.addClass('text-danger')
}


function clearErrors() {
    var email = $('#email');
    var password = $('#password');
    clearElementError(email);
    clearElementError(password);
    $('#errorMessage').text('');
}



function signup() {

    var email = $('#email');
    var password = $('#password');
    var rememberMe = $('#rememberMe');

    var data = {
        email:email.val(),
        password:password.val(),
        rememberMe:rememberMe.val(),
        landingPage:'http://viantinc.com/'
    };

    var hasError = false;
    if(!checkIfEmpty(email)) {
        hasError = true;
    }
    if(!checkIfEmpty(password)) {
        hasError = true;
    }
    if(hasError) {
        return;
    }

    $.ajax({
        type: "POST",
        url: "/api/signin/",
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
                errorSource = data.errorSource;
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
    // $('#password').val('abc');
});





function setElementError(element, message) {
    var small = element.parent().find('small').first();
    if(message.indexOf("$label") !==-1) {
        var label = element.parent().parent().find('label').text()
        message = message.replace("$label", label)
    }
    small.text(message);
    small.addClass('text-danger');
}

function clearElementError(element, message) {
    var small = element.parent().find('small').first();
    small.removeClass('text-danger');
    small.text();
}


