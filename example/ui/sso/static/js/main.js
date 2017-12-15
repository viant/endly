
function checkIfEmpty(element) {
    if (element.val() === '') {
        setElementError(element, "$label can not be empty")
        return false;
    }
    clearElementError(element);
    return true
}
