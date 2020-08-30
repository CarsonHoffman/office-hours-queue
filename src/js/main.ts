import { QueueApplication, User, CreateCourseDialog, CreateQueueDialog, EditStaffDialog } from './QueueApplication';
import { Schedule, ManageQueueDialog, OrderedQueue } from './OrderedQueue';
import $ from 'jquery';
import 'bootstrap3/dist/js/bootstrap.js';
import 'bootstrap3/dist/css/bootstrap.css';
import {
    AppointmentsQueue,
    AppointmentsSchedulePicker,
} from './AppointmentsQueue';

// import {gapi} from "https://apis.google.com/js/platform.js";

export function onSignIn(googleUser: gapi.auth2.GoogleUser) {
    var profile = googleUser.getBasicProfile();
    //        console.log('Name: ' + profile.getName());
    //        console.log('Image URL: ' + profile.getImageUrl());
    //        console.log('ID: ' + profile.getId()); // Do not send to your backend! Use an ID token instead.
    User.signIn(profile.getEmail(), googleUser.getAuthResponse().id_token);
    //        console.log(googleUser.getAuthResponse().expires_at());
    //        var expires_at = googleUser.getAuthResponse().expires_at;
    //        var now = Date.now();
    //        var now2 = Date.now();
}

function setupDialogs() {
    var clearInput = $('#clearInput');
    var clearTheQueueDialog = $('#clearTheQueueDialog');
    clearTheQueueDialog.on('shown.bs.modal', function () {
        clearInput.focus();
    });
    clearTheQueueDialog.on('show.bs.modal', function () {
        clearInput.val('');
    });
    clearInput.on('input', function (e) {
        if ($(this).val() == 'clear') {
            clearTheQueueDialog.modal('hide');
            let aq = QueueApplication.instance.activeQueue();
            aq && aq.queue instanceof OrderedQueue && aq.queue.clear();
        }
    });

    var signUpDialog = $('#signUpDialog');
    signUpDialog.on('show.bs.modal', function () {
        $(this).find('input').val('');
    });
    signUpDialog.on('shown.bs.modal', function () {
        $(this).find('input:first').focus();
    });

    let sendMessageDialog = $('#sendMessageDialog');
    sendMessageDialog.on('show.bs.modal', function () {
        $(this).find('input').val('');
    });
    sendMessageDialog.on('shown.bs.modal', function () {
        $(this).find('input:first').focus();
    });

    let sendMessageForm = $('#sendMessageForm');
    sendMessageForm.submit(function (e) {
        e.preventDefault();
        let content: string = <string>$('#sendMessageContent').val();

        if (!content || content.length == 0) {
            alert("You can't send a blank message.");
            return false;
        }

        QueueApplication.instance.sendMessage(content);

        sendMessageDialog.modal('hide');
        return false;
    });

    let addAnnouncementDialog = $('#addAnnouncementDialog');
    addAnnouncementDialog.on('show.bs.modal', function () {
        $(this).find('textarea').val('');
    });
    addAnnouncementDialog.on('shown.bs.modal', function () {
        $(this).find('textarea').focus();
    });

    let addAnnouncementForm = $('#addAnnouncementForm');
    addAnnouncementForm.submit((e) => {
        e.preventDefault();
        let content = <string>$('#addAnnouncementContent').val();

        if (!content || content.length == 0) {
            alert("You can't post a blank announcement.");
            return false;
        }

        let aq = QueueApplication.instance.activeQueue();
        aq && aq.addAnnouncement(content);

        addAnnouncementDialog.modal('hide');
        return false;
    });

    new Schedule($('#schedulePicker'));

    new ManageQueueDialog();
    new CreateCourseDialog();
    new CreateQueueDialog();
    new EditStaffDialog();

    let removeMyAppointmentInput = $('#removeMyAppointmentInput');
    let removeMyAppointmentDialog = $('#removeMyAppointmentDialog');
    removeMyAppointmentDialog.on('shown.bs.modal', function () {
        removeMyAppointmentInput.focus();
    });
    removeMyAppointmentDialog.on('show.bs.modal', function () {
        removeMyAppointmentInput.val('');
    });
    removeMyAppointmentInput.on('input', function (e) {
        if ($(this).val() == 'cancel') {
            removeMyAppointmentDialog.modal('hide');
            let aq = QueueApplication.instance.activeQueue();
            aq &&
                aq.queue instanceof AppointmentsQueue &&
                aq.queue.myRequest &&
                aq.queue.removeAppointment(aq.queue.myRequest);
        }
    });

    new AppointmentsSchedulePicker();
}

$(document).ready(function () {
    QueueApplication.createInstance($('#queueApplication'));
    // User.setTarget(UnauthenticatedUser.instance());

    setupDialogs();

    // <div class="g-signin2" data-onsuccess="onSignIn"></div>
    // Recurring refresh
    //setInterval(function(){
    //    QueueApplication.refreshActiveQueue();
    //}, 60000);
    // MOVED TO USER CODE IN queue.js
});

console.log('TEST BLAH');
