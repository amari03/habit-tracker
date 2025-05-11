// Script for sidebar navigation
const openNavButton = document.getElementById('openNav');
const closeNavButton = document.getElementById('closeNav');
const sidebar = document.getElementById('sidebar');
const overlay = document.getElementById('overlay');

if (openNavButton) { // Check if the open button exists (it might not on auth pages)
    openNavButton.addEventListener('click', function() {
        sidebar.classList.add('open');
        overlay.classList.add('active');
    });
}

if (closeNavButton) { // Check if the close button exists
    closeNavButton.addEventListener('click', function() {
        sidebar.classList.remove('open');
        overlay.classList.remove('active');
    });
}

if (overlay) { // Check if the overlay exists
    overlay.addEventListener('click', function() {
        sidebar.classList.remove('open');
        overlay.classList.remove('active');
    });
}

// Add form validation for all forms
document.addEventListener('DOMContentLoaded', function() {
    // Function to validate a form
    function validateForm(form, e) {
        let isValid = true;
        
        // Function to validate a field
        function validateField(field, errorMessage) {
            if (!field) return true; // If field doesn't exist, skip validation for it

            if (!field.value.trim()) {
                // Add invalid class to the field
                field.classList.add('invalid');
                
                // Add error message if it doesn't exist
                let errorDiv = field.parentNode.querySelector('.error');
                if (!errorDiv) {
                    errorDiv = document.createElement('div');
                    errorDiv.className = 'error';
                    field.parentNode.appendChild(errorDiv);
                }
                errorDiv.textContent = errorMessage;
                return false;
            } else {
                // Remove invalid class and error message if field is valid
                field.classList.remove('invalid');
                const errorDiv = field.parentNode.querySelector('.error');
                if (errorDiv) {
                    errorDiv.remove();
                }
                return true;
            }
        }
        
        // Get the fields
        const titleField = form.querySelector('#title');
        const descriptionField = form.querySelector('#description'); // This was likely the culprit for console errors
        const goalField = form.querySelector('#goal');
        
        // Validate each field
        // Only prevent default if any specific validation actually fails
        let formIsValid = true;
        if (titleField && !validateField(titleField, 'must be provided')) formIsValid = false;
        if (descriptionField && !validateField(descriptionField, 'must be provided')) formIsValid = false;
        if (goalField && !validateField(goalField, 'must be provided')) formIsValid = false;
        
        if (!formIsValid) {
            e.preventDefault(); // Prevent submission only if validation fails
        }
        
        return formIsValid;
    }
    
    // Add validation to the habit creation form
    const habitForm = document.getElementById('habit-form');
    if (habitForm) {
        habitForm.addEventListener('submit', function(e) {
            validateForm(this, e); // Call validateForm
        });
    }
    
    // Add validation to the edit form
    const editForm = document.querySelector('.edit-form');
    if (editForm) {
        editForm.addEventListener('submit', function(e) {
            validateForm(this, e); // Call validateForm
        });
    }
});

// Verify HTMX is loaded
if (typeof htmx !== 'undefined') {
console.log('HTMX version:', htmx.version);
} else {
console.error('HTMX not loaded!');
}