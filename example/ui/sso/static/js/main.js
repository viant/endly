function checkIfEmpty(element) {
    if (element.val() === '') {
        setError(element, "$label can not be empty")
        return false;
    }
    clearError(element);
    return true
}

function validateEmail(email) {
    var re = /^(([^<>()[\]\\.,;:\s@\"]+(\.[^<>()[\]\\.,;:\s@\"]+)*)|(\".+\"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$/;
    return re.test(email);
}

function signup() {

    var email = $('#email');
    var name = $('#name');
    var password = $('#password');
    var retypedPassword = $('#retypedPassword');
    var dataOfBirth = $('#dataOfBirth');
    var data = {

    };
    var hasError = false;
    if(checkIfEmpty(email)) {
        if(!validateEmail(email.val())) {
            setError(email, "$label does not look like a valid email")
            hasError = true;
        } else {
            clearError(email);
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
            clearError(retypedPassword);
        }
    }
    if(!checkIfEmpty(dataOfBirth)) {
        hasError = true;
    }
    if(hasError) {
        return;
    }
    $.ajax({
        type: "POST",
        url: "/api/singup/",
        data: data,
        success: function (data) {
            console.log('success');
            console.log(data);
        }
    });


}


$(document).ready(function () {
    $("form").submit(function (e) {
        e.preventDefault();
        signup();
    });
    $('#email').val('abc@ew.pl');
    $('#name').val('asds');
    $('#password').val('abc');
    $('#retypedPassword').val('abc');
    $('#dataOfBirth').val('1987-04-02')

});




function setError(element, message) {
    element.addClass('form-control-danger');
    element.parent().addClass('has-danger');
    var small = element.parent().find('small');
    if(message.indexOf("$label") !=-1) {
        var label = element.parent().parent().find('label').text()
        message = message.replace("$label", label)
    }
    small.text(message)
}

function clearError(element, message) {
    element.removeClass('form-control-danger');
    element.parent().removeClass('has-danger')
    element.removeClass('form-control-success');
    element.parent().addClass('has-success');
    small = element.parent().find('small');
    small.text();
}


