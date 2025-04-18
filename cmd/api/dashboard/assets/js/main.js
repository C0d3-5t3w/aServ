let token = localStorage.getItem('token');
let currentUser = JSON.parse(localStorage.getItem('currentUser'));


const api = {
    baseUrl: '/api',
    
    async request(endpoint, options = {}) {
        const headers = {
            'Content-Type': 'application/json',
            ...options.headers
        };
        
        if (token) {
            headers['Authorization'] = `Bearer ${token}`;
        }
        
        const response = await fetch(`${this.baseUrl}${endpoint}`, {
            ...options,
            headers
        });
        
        const data = await response.json();
        
        if (!response.ok) {
            throw new Error(data.error || 'Something went wrong');
        }
        
        return data;
    },
    
    async login(username, password) {
        const result = await this.request('/auth/login', {
            method: 'POST',
            body: JSON.stringify({ username, password })
        });
        
        token = result.data.token;
        currentUser = {
            id: result.data.user_id,
            username: result.data.username
        };
        
        localStorage.setItem('token', token);
        localStorage.setItem('currentUser', JSON.stringify(currentUser));
        
        return result;
    },
    
    async register(username, password, email) {
        return await this.request('/auth/register', {
            method: 'POST',
            body: JSON.stringify({ username, password, email })
        });
    },
    
    async getUsers() {
        return await this.request('/users');
    },
    
    async getUser(id) {
        return await this.request(`/users/${id}`);
    },
    
    async getItems() {
        return await this.request('/items');
    },
    
    async getItem(id) {
        return await this.request(`/items/${id}`);
    },
    
    async createItem(item) {
        return await this.request('/items', {
            method: 'POST',
            body: JSON.stringify(item)
        });
    },
    
    async updateItem(id, item) {
        return await this.request(`/items/${id}`, {
            method: 'PUT',
            body: JSON.stringify(item)
        });
    },
    
    async deleteItem(id) {
        return await this.request(`/items/${id}`, {
            method: 'DELETE'
        });
    },
    
    logout() {
        localStorage.removeItem('token');
        localStorage.removeItem('currentUser');
        token = null;
        currentUser = null;
        navigateTo('login');
    }
};


function showPage(pageId) {
    document.querySelectorAll('.page').forEach(page => {
        page.style.display = 'none';
    });
    
    document.getElementById(pageId).style.display = 'block';
}


function navigateTo(pageId) {
    if (!token && !['login', 'register'].includes(pageId)) {
        pageId = 'login';
    }
    
    showPage(pageId);
    
    
    document.querySelectorAll('nav a').forEach(link => {
        if (link.getAttribute('data-page') === pageId) {
            link.classList.add('active');
        } else {
            link.classList.remove('active');
        }
    });
    
    
    switch (pageId) {
        case 'dashboard':
            loadDashboard();
            break;
        case 'users':
            loadUsers();
            break;
        case 'items':
            loadItems();
            break;
    }
}


document.addEventListener('DOMContentLoaded', () => {
    
    document.querySelectorAll('nav a').forEach(link => {
        link.addEventListener('click', (e) => {
            e.preventDefault();
            navigateTo(link.getAttribute('data-page'));
        });
    });
    
    
    document.getElementById('login-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        const username = document.getElementById('login-username').value;
        const password = document.getElementById('login-password').value;
        
        try {
            await api.login(username, password);
            showMessage('login-message', 'Login successful!', 'success');
            navigateTo('dashboard');
        } catch (error) {
            showMessage('login-message', error.message, 'error');
        }
    });
    
    
    document.getElementById('register-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        const username = document.getElementById('register-username').value;
        const password = document.getElementById('register-password').value;
        const email = document.getElementById('register-email').value;
        
        try {
            await api.register(username, password, email);
            showMessage('register-message', 'Registration successful! Please login.', 'success');
            document.getElementById('register-form').reset();
            setTimeout(() => navigateTo('login'), 2000);
        } catch (error) {
            showMessage('register-message', error.message, 'error');
        }
    });
    
    
    document.getElementById('add-item-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        const name = document.getElementById('item-name').value;
        const description = document.getElementById('item-description').value;
        const price = parseFloat(document.getElementById('item-price').value);
        
        try {
            await api.createItem({ name, description, price });
            showMessage('items-message', 'Item added successfully!', 'success');
            document.getElementById('add-item-form').reset();
            loadItems();
        } catch (error) {
            showMessage('items-message', error.message, 'error');
        }
    });
    
    
    document.addEventListener('click', async (e) => {
        
        if (e.target.classList.contains('delete-item-btn')) {
            const itemId = e.target.getAttribute('data-id');
            if (confirm('Are you sure you want to delete this item?')) {
                try {
                    await api.deleteItem(itemId);
                    showMessage('items-message', 'Item deleted successfully!', 'success');
                    loadItems();
                } catch (error) {
                    showMessage('items-message', error.message, 'error');
                }
            }
        }
        
        
        if (e.target.classList.contains('edit-item-btn')) {
            const itemId = e.target.getAttribute('data-id');
            try {
                const result = await api.getItem(itemId);
                const item = result.data;
                
                
                document.getElementById('edit-item-id').value = item.id;
                document.getElementById('edit-item-name').value = item.name;
                document.getElementById('edit-item-description').value = item.description;
                document.getElementById('edit-item-price').value = item.price;
                
                
                document.getElementById('edit-modal').style.display = 'block';
            } catch (error) {
                showMessage('items-message', error.message, 'error');
            }
        }
    });
    
    
    document.getElementById('edit-item-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        const id = document.getElementById('edit-item-id').value;
        const name = document.getElementById('edit-item-name').value;
        const description = document.getElementById('edit-item-description').value;
        const price = parseFloat(document.getElementById('edit-item-price').value);
        
        try {
            await api.updateItem(id, { name, description, price });
            document.getElementById('edit-modal').style.display = 'none';
            showMessage('items-message', 'Item updated successfully!', 'success');
            loadItems();
        } catch (error) {
            showMessage('items-message', error.message, 'error');
        }
    });
    
    
    document.querySelectorAll('.close-modal').forEach(btn => {
        btn.addEventListener('click', () => {
            document.getElementById('edit-modal').style.display = 'none';
        });
    });
    
    
    document.getElementById('logout-btn').addEventListener('click', () => {
        api.logout();
    });
    
    
    if (token) {
        navigateTo('dashboard');
        updateUserInfo();
    } else {
        navigateTo('login');
    }
});


function showMessage(elementId, message, type) {
    const element = document.getElementById(elementId);
    element.textContent = message;
    element.className = `alert alert-${type}`;
    element.style.display = 'block';
    
    setTimeout(() => {
        element.style.display = 'none';
    }, 5000);
}

function updateUserInfo() {
    if (currentUser) {
        document.getElementById('current-user').textContent = currentUser.username;
    }
}


async function loadDashboard() {
    try {
        const stats = {
            users: (await api.getUsers()).data.length,
            items: (await api.getItems()).data.length
        };
        
        document.getElementById('stats-users').textContent = stats.users;
        document.getElementById('stats-items').textContent = stats.items;
    } catch (error) {
        showMessage('dashboard-message', error.message, 'error');
    }
}

async function loadUsers() {
    try {
        const result = await api.getUsers();
        const users = result.data;
        const tbody = document.getElementById('users-table-body');
        tbody.innerHTML = '';
        
        users.forEach(user => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td>${user.username}</td>
                <td>${user.email}</td>
                <td>${new Date(user.created_at).toLocaleDateString()}</td>
            `;
            tbody.appendChild(row);
        });
    } catch (error) {
        showMessage('users-message', error.message, 'error');
    }
}

async function loadItems() {
    try {
        const result = await api.getItems();
        const items = result.data;
        const tbody = document.getElementById('items-table-body');
        tbody.innerHTML = '';
        
        items.forEach(item => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td>${item.name}</td>
                <td>${item.description}</td>
                <td>$${item.price.toFixed(2)}</td>
                <td>${new Date(item.created_at).toLocaleDateString()}</td>
                <td>
                    <button class="edit-item-btn" data-id="${item.id}">Edit</button>
                    <button class="delete-item-btn" data-id="${item.id}">Delete</button>
                </td>
            `;
            tbody.appendChild(row);
        });
    } catch (error) {
        showMessage('items-message', error.message, 'error');
    }
}
