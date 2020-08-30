import $ from 'jquery';

declare global {
    interface Array<T> {
        clear(): void;
    }
}
Array.prototype.clear = function () {
    this.length = 0;
};

export type Mutable<T> = { -readonly [P in keyof T]: T[P] };

export function asMutable<T>(obj: T): Mutable<T> {
    return <Mutable<T>>obj;
}

function debug(message: string, category: string) {
    if (category) {
        console.log(category + ': ' + message);
        $('.debug.' + category).html('' + message); //""+ is to force conversion to string (via .toString if object)
    } else {
        console.log(message);
        $('.debug.debugAll').html('' + message); //""+ is to force conversion to string (via .toString if object)
    }
}

export function assert(
    condition: any,
    message: string = '',
): asserts condition {
    if (!condition) {
        throw Error('Assert failed: ' + message);
    }
}

export function assertFalse(message: string = ''): never {
    throw Error('Assert failed: ' + message);
}

export function oops(xhr: any, textStatus?: any) {
    if (textStatus === 'abort') {
        return;
    }
    console.log('Oops. An error occurred. Try refreshing the page.');

    if (xhr) {
        showErrorMessage(JSON.parse(xhr.responseText)['message']);
        return;
    }

    $('#oopsDialog').modal('show');
}

export function showErrorMessage(message: any) {
    if (message.message) {
        showErrorMessage(message.message);
        return;
    }
    console.log(message);
    $('#errorMessage').html(message);
    $('#errorDialog').modal('show');
}
