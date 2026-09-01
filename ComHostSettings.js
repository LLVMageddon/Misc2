/*
Beware:
A stray alert() like this, seen by the wrong person with a pre-existing bias and zero understanding of how code is 
developed and managed, can be transformed into a weapon. It can easily be used to portray you as unprofessional, 
careless, and unfit for the job, even if it was just a temporary, debugging related artifact, that never even 
left Devel or made its way to Code Review, QA or Prod.

Notice there is no implementation detail.
*/

$(document).ready(function() {
    //look at me, I'm using JQuery!
    //id of button can't be changed, else stuff breaks. Just no!!
    $("#SetingsComitButton").off('click').on('click', function() {
        alert("Fcuk This!, I hate JS.");//quick-&-dirty debugging in older CB
        /*fancy footwork here*/
    });
});
