# Configuration supplémentaire des machines virtuelles (extra_config)

Le provider CloudTemple permet maintenant de configurer des paramètres avancés sur les machines virtuelles via le champ `extra_config`. Cette fonctionnalité est particulièrement utile pour configurer des systèmes d'exploitation spécialisés comme CoreOS avec Ignition.

## Configurations supportées

### Ignition pour CoreOS

Les configurations suivantes sont spécifiquement supportées pour l'intégration avec Ignition et CoreOS :

- `guestinfo.ignition.config.data` : Données de configuration Ignition (généralement encodées en base64)
- `guestinfo.ignition.config.data.encoding` : Type d'encodage utilisé pour les données Ignition
- `guestinfo.afterburn.initrd.network-kargs` : Arguments réseau pour Afterburn
- `stealclock.enable` : Active le steal clock pour les environnements virtualisés

## Exemple d'utilisation

```hcl
resource "cloudtemple_compute_virtual_machine" "coreos_vm" {
  name                = "coreos-ignition-vm"
  datacenter_id       = var.datacenter_id
  host_cluster_id     = var.host_cluster_id
  datastore_id        = var.datastore_id
  
  memory = 2147483648  # 2GB
  cpu    = 2
  
  guest_operating_system_moref = var.coreos_guest_os_moref
  power_state = "on"
  
  # Configurations supplémentaires
  extra_config = {
    # Configuration Ignition encodée en base64
    "guestinfo.ignition.config.data" = base64encode(jsonencode({
      ignition = {
        version = "3.3.0"
      }
      passwd = {
        users = [{
          name = "core"
          ssh_authorized_keys = [
            "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC..."
          ]
        }]
      }
      systemd = {
        units = [{
          name = "docker.service"
          enabled = true
        }]
      }
    }))
    
    "guestinfo.ignition.config.data.encoding" = "base64"
    "guestinfo.afterburn.initrd.network-kargs" = "ip=dhcp"
    "stealclock.enable" = "TRUE"
  }
}
```

## Configuration Ignition détaillée

### Structure de base

```json
{
  "ignition": {
    "version": "3.3.0"
  },
  "passwd": {
    "users": [
      {
        "name": "core",
        "ssh_authorized_keys": [
          "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC..."
        ]
      }
    ]
  },
  "systemd": {
    "units": [
      {
        "name": "docker.service",
        "enabled": true
      }
    ]
  },
  "storage": {
    "files": [
      {
        "path": "/etc/hostname",
        "mode": 420,
        "contents": {
          "source": "data:,my-coreos-host"
        }
      }
    ]
  }
}
```

### Encodage et transmission

1. **Création de la configuration** : Créez votre configuration Ignition au format JSON
2. **Encodage** : Encodez la configuration en base64 avec `base64encode(jsonencode(...))`
3. **Transmission** : Passez la configuration encodée via `guestinfo.ignition.config.data`
4. **Spécification de l'encodage** : Indiquez `base64` dans `guestinfo.ignition.config.data.encoding`

## Configurations réseau avec Afterburn

Afterburn peut être configuré pour gérer automatiquement la configuration réseau au démarrage :

```hcl
extra_config = {
  "guestinfo.afterburn.initrd.network-kargs" = "ip=dhcp"
  # ou pour une configuration statique :
  # "guestinfo.afterburn.initrd.network-kargs" = "ip=192.168.1.100::192.168.1.1:255.255.255.0:hostname:eth0:none"
}
```

## Optimisations pour la virtualisation

### Steal Clock

Le steal clock doit être activé dans les environnements virtualisés pour une meilleure gestion du temps :

```hcl
extra_config = {
  "stealclock.enable" = "TRUE"
}
```

## Bonnes pratiques

1. **Validation** : Validez toujours votre configuration Ignition avant de l'encoder
2. **Sécurité** : Ne stockez jamais de secrets en clair dans la configuration
3. **Versioning** : Utilisez la version Ignition appropriée pour votre distribution CoreOS
4. **Test** : Testez vos configurations sur un environnement de développement avant la production

## Limitations

- Les modifications des `extra_config` nécessitent un redémarrage de la machine virtuelle pour être prises en compte
- Certaines configurations peuvent nécessiter des privilèges spécifiques sur l'hyperviseur
- La taille maximale des données de configuration peut être limitée par l'hyperviseur

## Dépannage

### Configuration Ignition non appliquée

1. Vérifiez que l'encodage base64 est correct
2. Validez la syntaxe JSON de votre configuration Ignition
3. Consultez les logs de démarrage de CoreOS pour les erreurs Ignition

### Problèmes réseau avec Afterburn

1. Vérifiez la syntaxe des arguments réseau
2. Assurez-vous que l'interface réseau spécifiée existe
3. Consultez les logs systemd pour les erreurs Afterburn

## Ressources supplémentaires

- [Documentation officielle Ignition](https://coreos.github.io/ignition/)
- [Guide Afterburn](https://coreos.github.io/afterburn/)
- [Spécifications CoreOS](https://docs.fedoraproject.org/en-US/fedora-coreos/)
