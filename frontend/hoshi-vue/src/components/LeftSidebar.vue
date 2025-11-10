<template>
  <nav class="sidebar">
    <div class="sidebar-content">
      <h1 class="logo">
        hoshiBmaTchi
      </h1>
      
      <ul class="nav-links">
        <li><router-link to="/" class="nav-item">Home</router-link></li>
        <li><button class="nav-item">Search</button></li> <li><router-link to="/explore" class="nav-item">Explore</router-link></li>
        <li><router-link to="/reels" class="nav-item">Reels</router-link></li>
        <li><router-link to="/messages" class="nav-item">Messages</router-link></li>
        <li><button class="nav-item">Notifications</button></li> <li><button class="nav-item">Create</button></li> <li><router-link to="/profile" class="nav-item">Profile</router-link></li>
      </ul>
    
      <div class="nav-footer">
        <button class="nav-item">More</button> </div>
    </div>
  </nav>
</template>

<style scoped>
.sidebar {
  width: 244px; /* Standard Insta width */
  height: 100vh; /* Full viewport height */
  position: fixed; /* Fixed to the left side */
  top: 0;
  left: 0;
  border-right: 1px solid rgb(54, 54, 54);
  padding: 8px 12px 20px;
  display: flex;
  flex-direction: column;
}

.sidebar-content {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.logo {
  font-size: 1.8rem;
  padding: 25px 12px 16px;
  margin-bottom: 1rem;
}

.nav-links {
  list-style: none;
  padding: 0;
  margin: 0;
  flex-grow: 1; /* This pushes the footer to the bottom */
}

.nav-footer {
  flex-shrink: 0; /* Prevents footer from shrinking */
}

/* Base styles for all nav items (links or buttons) */
.nav-item {
  display: block;
  width: 100%;
  padding: 12px;
  margin: 4px 0;
  font-size: 1rem;
  font-weight: 500;
  text-decoration: none;
  color: #fff;
  background: none;
  border: none;
  text-align: left;
  cursor: pointer;
  border-radius: 8px;
}

.nav-item:hover {
  background-color: rgba(255, 255, 255, 0.1);
}

/* This highlights the active page link [cite: 289] */
.router-link-active {
  font-weight: bold;
}
</style>