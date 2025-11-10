<template>
  <nav class="sidebar">
    <div class="sidebar-header">
      <h2>hoshiBmaTchi</h2>
    </div>
    
    <ul class="nav-links">
      <li><router-link to="/">Home</router-link></li> <li><router-link to="/explore">Explore</router-link></li> <li><router-link to="/reels">Reels</router-link></li> <li><router-link to="/messages">Messages</router-link></li> <li><router-link to="/profile">Profile</router-link></li> </ul>
    
    <div class="sidebar-actions">
      <button>Search</button> <button>Notifications</button> <button>Create</button> </div>
    
    <div class="sidebar-footer">
      <button>More</button>
    </div>
  </nav>
</template>

<style scoped>
/* This is critical. We give the sidebar a fixed height
   and make ITSELF scrollable if the content overflows. */
.sidebar {
  display: flex;
  flex-direction: column;
  width: 250px; /* Example width */
  height: 100vh; /* Full viewport height */
  position: fixed; /* Fix it to the left */
  top: 0;
  left: 0;
  border-right: 1px solid #333;
  padding: 1rem;
  
  /* This enables internal scrolling if nav links exceed height */
  overflow-y: auto; /*  */
}

.nav-links {
  list-style: none;
  padding: 0;
  margin: 2rem 0;
  flex-grow: 1; /* Pushes footer down */
}

.nav-links li {
  margin-bottom: 1rem;
}

/* TODO: Add styles for active links [cite: 723] */
.router-link-active {
  font-weight: bold;
}
</style>