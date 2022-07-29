module ApplicationHelper
  def flash_color_for(name)
    case name.to_sym
    when :alert
      'bg-danger bg-opacity-25'
    else
      'bg-success bg-opacity-25'
    end
  end

  def flash_icon_for(name)
    case name.to_sym
    when :alert
      'bi bi-exclamation-triangle text-danger'
    else
      'bi bi-bell text-success'
    end
  end
end
